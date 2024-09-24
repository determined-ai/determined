package db

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// AddTrial adds the trial to the database and sets its ID.
func AddTrial(ctx context.Context, trial *model.Trial, taskID model.TaskID) error {
	if trial.ID != 0 {
		return errors.Errorf("error adding a trial with non-zero id %v", trial.ID)
	}

	err := Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		run, v2, err := trialToRunAndTrialV2(ctx, tx, trial)
		if err != nil {
			return fmt.Errorf("converting trial to run and trialv2: %w", err)
		}

		var key string
		var localID int
		if err := tx.NewUpdate().Table("projects").
			Set("max_local_id = max_local_id + 1").Where("id = ?", run.ProjectID).
			Returning("key, max_local_id").Scan(ctx, &key, &localID); err != nil {
			return fmt.Errorf("updating and returning project max_local_id: %w", err)
		}
		run.LocalID = localID

		if _, err := tx.NewInsert().Model(run).Returning("id").Exec(ctx); err != nil {
			return fmt.Errorf("inserting trial run model: %w", err)
		}

		v2.RunID = run.ID
		if _, err := tx.NewInsert().Model(v2).Exec(ctx); err != nil {
			return fmt.Errorf("inserting trial v2 model: %w", err)
		}

		redirect := struct {
			bun.BaseModel `bun:"table:local_id_redirect"`

			RunID      int    `db:"run_id" bun:"run_id"`
			ProjectID  int    `db:"project_id" bun:"project_id"`
			ProjectKey string `db:"project_key" bun:"project_key"`
			LocalID    int    `db:"local_id" bun:"local_id"`
		}{
			RunID:      run.ID,
			ProjectID:  run.ProjectID,
			ProjectKey: key,
			LocalID:    localID,
		}
		if _, err := tx.NewInsert().Model(&redirect).Exec(ctx); err != nil {
			return fmt.Errorf("storing run_id in redirect table: %w", err)
		}

		trial.ID = run.ID // We need to mutate trial.ID.

		runTaskID := &model.RunTaskID{RunID: trial.ID, TaskID: taskID}
		if _, err := tx.NewInsert().Model(runTaskID).Exec(ctx); err != nil {
			return fmt.Errorf("inserting trial task id relationship: %w", err)
		}

		hparams, projHparams, err := BuildRunHParams(run.ID, run.ProjectID, run.HParams, "")
		if err != nil {
			return fmt.Errorf("getting run hyperparameters: %w", err)
		}

		if len(hparams) > 0 {
			if err := tx.NewInsert().Model(&hparams).Scan(ctx); err != nil {
				return fmt.Errorf("inserting run hyperparameters: %w", err)
			}
		}

		if len(projHparams) > 0 {
			if err := tx.NewInsert().Model(&projHparams).
				On("CONFLICT (project_id, hparam, type) DO NOTHING").Scan(ctx); err != nil {
				return fmt.Errorf("inserting project hyperparameters: %w", err)
			}
		}

		var isSingleTrial bool
		err = tx.NewSelect().
			ColumnExpr("config->'searcher'->>'name' = 'single'").
			Table("experiments").
			Where("id = ?", run.ExperimentID).
			Scan(ctx, &isSingleTrial)
		if err != nil {
			return fmt.Errorf("getting experiment config while inserting trial: %w", err)
		}
		if isSingleTrial {
			if _, err := tx.NewUpdate().Table("experiments").Set("best_trial_id = ?", run.ID).
				Where("id = ?", run.ExperimentID).Exec(ctx); err != nil {
				return fmt.Errorf("updating best trial id for single trial experiment: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("inserting trial %v: %w", trial, err)
	}

	return nil
}

// AddProjectHparams adds project hyperparams from provided runs to provided project.
func AddProjectHparams(ctx context.Context, tx bun.Tx, projectID int, runIDs []int32) error {
	if _, err := tx.NewRaw(`
		INSERT INTO project_hparams
		SELECT ?::int as project_id, hparam,
		CASE
			WHEN number_val iS NOT NULL THEN 'number'
			WHEN text_val iS NOT NULL THEN 'string'
			WHEN bool_val iS NOT NULL THEN 'boolean'
		END as type 
		FROM run_hparams WHERE run_id IN (?)
		GROUP BY hparam, type
		ON CONFLICT (project_id, hparam, type) DO NOTHING`,
		projectID, bun.In(runIDs)).
		Exec(ctx); err != nil {
		return fmt.Errorf("bulk inserting project hyperparameters: %w", err)
	}
	return nil
}

// RemoveOutdatedProjectHparams removes outdated project hyperparams from provided project.
func RemoveOutdatedProjectHparams(ctx context.Context, tx bun.Tx, projectID int) error {
	if _, err := tx.NewRaw(`
	WITH removed_project_hparams as 
	(SELECT * FROM project_hparams WHERE project_id=?
	EXCEPT
	SELECT ? as project_id, rhp.hparam, CASE
							WHEN rhp.number_val iS NOT NULL THEN 'number'
							WHEN rhp.text_val iS NOT NULL THEN 'string'
							WHEN rhp.bool_val iS NOT NULL THEN 'boolean'
					END as type FROM run_hparams as rhp JOIN
					(SELECT id, project_id FROM runs WHERE project_id=?) as r ON
					r.id=rhp.run_id GROUP BY hparam, type)
	DELETE FROM project_hparams as p WHERE EXISTS
	(SELECT * FROM removed_project_hparams as rm
		 WHERE (p.project_id=rm.project_id AND p.type=rm.type AND p.hparam=rm.hparam))
	`, projectID, projectID, projectID).Exec(ctx); err != nil {
		return fmt.Errorf("bulk deleting project hyperparameters: %w", err)
	}
	return nil
}

// BuildRunHParams builds hyperparameters objects to add into the `run_hparams` & `project_hparams` table.
func BuildRunHParams(runID int, projectID int, hparams map[string]any,
	parentName string,
) ([]model.RunHparam, []model.ProjectHparam, error) {
	hparamsModel := []model.RunHparam{}
	projHparamsModel := []model.ProjectHparam{}
	for hpName, v := range hparams {
		hp := model.RunHparam{
			RunID:  runID,
			HParam: parentName + hpName,
		}
		projHp := model.ProjectHparam{
			ProjectID: projectID,
			HParam:    parentName + hpName,
		}
		switch val := v.(type) {
		case float64:
			hp.NumberVal = &val
			projHp.Type = MetricTypeNumber
		case int:
			conv := float64(val)
			hp.NumberVal = &conv
			projHp.Type = MetricTypeNumber
		case string:
			hp.TextVal = &val
			projHp.Type = MetricTypeString
		case bool:
			hp.BoolVal = &val
			projHp.Type = MetricTypeBool
		case map[string]any:
			nestedHParams, nestedProjHparams, err := BuildRunHParams(runID, projectID, v.(map[string]any), hpName+".")
			if err != nil {
				return hparamsModel, projHparamsModel, fmt.Errorf("failed to get nested hyperperameters for %s: %w", hpName, err)
			}
			hparamsModel = append(hparamsModel, nestedHParams...)
			projHparamsModel = append(projHparamsModel, nestedProjHparams...)
			continue
		default:
			valBytes, err := json.Marshal(v)
			if err != nil {
				return hparamsModel, projHparamsModel,
					fmt.Errorf("cannot assign hyperparameter %s, failed to encode type %T: %w", hpName, val, err)
			}
			valString := string(valBytes)
			hp.TextVal = &valString
			projHp.Type = MetricTypeString
		}
		hparamsModel = append(hparamsModel, hp)
		projHparamsModel = append(projHparamsModel, projHp)
	}

	return hparamsModel, projHparamsModel, nil
}

// UpsertTrialByExternalIDTx UPSERTs the trial with respect to the external_trial_id.
func UpsertTrialByExternalIDTx(
	ctx context.Context, tx bun.Tx, trial *model.Trial, taskID model.TaskID,
) error {
	if trial.ID != 0 {
		return errors.Errorf("error adding a trial with non-zero id %v", trial.ID)
	}

	run, v2, err := trialToRunAndTrialV2(ctx, tx, trial)
	if err != nil {
		return fmt.Errorf("upsert converting trial to run and trialv2: %w", err)
	}

	if _, err := tx.NewInsert().Model(run).
		On("CONFLICT (experiment_id, external_run_id) DO UPDATE").
		Set("hparams = EXCLUDED.hparams").
		Returning("id").Exec(ctx); err != nil {
		return fmt.Errorf("upserting trial run model: %w", err)
	}

	v2.RunID = run.ID
	if _, err := tx.NewInsert().Model(v2).
		On("CONFLICT (run_id) DO NOTHING").
		Exec(ctx); err != nil {
		return fmt.Errorf("upserting trial v2 model: %w", err)
	}

	trial.ID = run.ID // We need to mutate trial.ID.

	runTaskID := &model.RunTaskID{RunID: run.ID, TaskID: taskID}
	if _, err := tx.NewInsert().Model(runTaskID).
		On("CONFLICT (run_id, task_id) DO NOTHING").Exec(ctx); err != nil {
		return fmt.Errorf("upserting trial task id relationship: %w", err)
	}

	return nil
}

// TrialByID looks up a trial by ID, returning an error if none exists.
func TrialByID(ctx context.Context, id int) (*model.Trial, error) {
	t := &model.Trial{}
	if err := Bun().NewSelect().Model(t).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, fmt.Errorf("error querying for trial %d: %w", id, err)
	}
	return t, nil
}

// TrialTaskIDsByTrialID returns trial id task ids by trial ID, sorted by start time.
func TrialTaskIDsByTrialID(ctx context.Context, trialID int) ([]*model.RunTaskID, error) {
	var ids []*model.RunTaskID
	if err := Bun().NewSelect().Model(&ids).
		Where("run_id = ?", trialID).
		Join("JOIN tasks t ON run_task_id.task_id = t.task_id").
		Order("t.start_time").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("getting tasks for trial ID %d: %w", trialID, err)
	}
	return ids, nil
}

// TrialIDByRequestID looks up a trial ID by request ID, returning an error if none exists.
// This is only used to shim legacy experiment snapshots.
func TrialIDByRequestID(
	ctx context.Context, requestID model.RequestID,
) (*int, error) {
	var trialID int
	if err := Bun().NewSelect().
		Where("request_id = ?", requestID).Scan(ctx, &trialID); err != nil {
		return nil, fmt.Errorf("error querying for trial %s: %w", requestID, err)
	}
	return &trialID, nil
}

// TrialByTaskID looks up a trial by taskID, returning an error if none exists.
// This errors if you called it with a non trial taskID.
func TrialByTaskID(ctx context.Context, taskID model.TaskID) (*model.Trial, error) {
	var t model.Trial
	if err := Bun().NewSelect().Model(&t).
		Where("tt.task_id = ?", taskID).
		Join("JOIN run_id_task_id tt ON trial.id = tt.run_id").
		Scan(ctx, &t); err != nil {
		return nil, fmt.Errorf("error querying for trial taskID %s: %w", taskID, err)
	}
	return &t, nil
}

// UpdateTrial updates the state of an existing trial.
// end_time is set if the trial moves to a terminal state.
func UpdateTrial(ctx context.Context, id int, newState model.State) error {
	trial, err := TrialByID(ctx, id)
	if err != nil {
		return fmt.Errorf("error finding trial %v to update: %w", id, err)
	}

	// Update trial state if necessary.
	if trial.State == newState {
		return nil
	}

	if !model.TrialTransitions[trial.State][newState] {
		return fmt.Errorf("illegal transition %v -> %v for trial %v",
			trial.State, newState, trial.ID)
	}
	toUpdate := []string{"state"}
	trial.State = newState
	if model.TerminalStates[newState] {
		now := time.Now().UTC()
		trial.EndTime = &now
		toUpdate = append(toUpdate, "end_time")
	}

	return Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		run, _, err := trialToRunAndTrialV2(ctx, tx, trial)
		if err != nil {
			return fmt.Errorf("update trial converting trial to run and trialv2: %w", err)
		}

		if _, err := tx.NewUpdate().Model(run).Column(toUpdate...).Where("id = ?", id).
			Exec(ctx); err != nil {
			return fmt.Errorf("error updating (%v) in trial %v: %w", strings.Join(toUpdate, ", "), id, err)
		}

		if model.TerminalStates[newState] && trial.EndTime != nil {
			if _, err := tx.NewRaw(`UPDATE tasks SET end_time = ? FROM run_id_task_id 
				WHERE run_id_task_id.task_id = tasks.task_id AND run_id_task_id.run_id = ? AND end_time IS NULL`,
				*trial.EndTime, id).Exec(ctx); err != nil {
				return fmt.Errorf("completing task: %w", err)
			}
		}

		return nil
	})
}

// UpdateTrialFields updates the specified fields of trial with ID id. Fields that are nil or zero
// are not updated.
func (db *PgDB) UpdateTrialFields(id int, newRunnerMetadata *trialv1.TrialRunnerMetadata, newRunID,
	newRestarts int,
) error {
	ctx := context.TODO()
	trial, err := TrialByID(ctx, id)
	if err != nil {
		return fmt.Errorf("error finding trial %v to update: %w", id, err)
	}

	var toUpdate []string

	// Update trial runner's state if necessary.
	if newRunnerMetadata != nil {
		trial.RunnerState = newRunnerMetadata.State
		toUpdate = append(toUpdate, "runner_state")
	}

	// Update trial's run id if necessary.
	if newRunID > 0 {
		trial.RunID = newRunID
		toUpdate = append(toUpdate, "restart_id")
	}

	// Update trial's restart count if necessary.
	if newRestarts > 0 {
		trial.Restarts = newRestarts
		toUpdate = append(toUpdate, "restarts")
	}

	run, _, err := trialToRunAndTrialV2(ctx, Bun(), trial)
	if err != nil {
		return fmt.Errorf("converting trial to run for update: %w", err)
	}

	_, err = Bun().NewUpdate().Model(run).Column(toUpdate...).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("updating trial fields: %w", err)
	}

	return err
}

// TrialRunIDAndRestarts returns the run id and restart count for a trial.
func (db *PgDB) TrialRunIDAndRestarts(trialID int) (runID int, restart int, err error) {
	if err := db.sql.QueryRowx(`
SELECT run_id, restarts
FROM trials
WHERE id = $1`, trialID).Scan(&runID, &restart); err != nil {
		return 0, 0, errors.Wrap(err, "failed to scan trial restart count")
	}
	return runID, restart, nil
}

// fullTrialSummaryMetricsRecompute recomputes all summary metrics for a given trial.
func (db *PgDB) fullTrialSummaryMetricsRecompute(
	ctx context.Context, tx *sqlx.Tx, trialID int,
) error {
	// TODO(DET-9566): we can probably limit this to recompute only a single metric type and it would
	// fit the current usage better.
	updatedSummaryMetrics := model.JSONObj{}
	metricGroups := []model.MetricGroup{}
	if err := tx.SelectContext(ctx, &metricGroups, `
SELECT DISTINCT metric_group FROM metrics WHERE partition_type = 'GENERIC' AND trial_id = $1
	`,
		trialID); err != nil {
		return err
	}
	metricGroups = append(metricGroups, model.TrainingMetricGroup)
	metricGroups = append(metricGroups, model.ValidationMetricGroup)

	for _, metricGroup := range metricGroups {
		summary, err := db.calculateFullTrialSummaryMetrics(
			ctx, tx, trialID, metricGroup)
		if err != nil {
			return fmt.Errorf("rollback computing %s summary metrics: %w", metricGroup, err)
		}
		if len(summary) > 0 {
			key := model.TrialSummaryMetricsJSONPath(metricGroup)
			updatedSummaryMetrics[key] = summary
		}
	}

	for k, v := range updatedSummaryMetrics {
		switch v := v.(type) {
		case model.JSONObj, map[string]any:
		default:
			log.Errorf("when full compute updating summary metric "+
				"%+v path %s type %T value %+v is not a map, setting to empty map",
				updatedSummaryMetrics,
				k,
				v,
				v,
			)
			updatedSummaryMetrics[k] = model.JSONObj{}
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE runs SET summary_metrics = $1,
	summary_metrics_timestamp = NOW() WHERE id = $2`, updatedSummaryMetrics, trialID); err != nil {
		return fmt.Errorf("rollback updating trial summary metrics: %w", err)
	}
	return nil
}

func (db *PgDB) calculateFullTrialSummaryMetrics(
	ctx context.Context, tx *sqlx.Tx, trialID int, mGroup model.MetricGroup,
) (model.JSONObj, error) {
	metricGroup := string(mGroup)
	partition := customMetricGroupToPartitionType(&metricGroup)
	jsonPath := model.TrialMetricsJSONPath(partition == ValidationMetric)
	//nolint: execinquery
	rows, err := tx.QueryContext(ctx, db.queries.GetOrLoad("calculate-full-trial-summary-metrics"),
		trialID, jsonPath, partition, mGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "getting full compute trial %d summary metrics", trialID)
	}

	metrics := model.JSONObj{}
	defer rows.Close()
	for rows.Next() {
		var metric model.JSONObj
		var name string
		if err = rows.Scan(&name, &metric); err != nil {
			return nil, fmt.Errorf("scanning summary metric row: %w", err)
		}
		metrics[name] = metric
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err summary metric full compute: %w", err)
	}

	return metrics, nil
}

// updateLatestValidationID updates latest validation based on validations table.
func (db *PgDB) updateLatestValidationID(ctx context.Context, tx *sqlx.Tx, trialID int) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE runs SET latest_validation_id = (
			SELECT validations.id
			FROM validations
			JOIN trials t ON validations.trial_id = t.id
			JOIN experiments e on t.experiment_id = e.id
			WHERE trial_id = $1 AND (
				validations.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric')
			) IS NOT NULL
			ORDER BY validations.end_time DESC
			LIMIT 1
		) WHERE id = $1`, trialID); err != nil {
		return fmt.Errorf("updating latest validation id for trial %d: %w", trialID, err)
	}
	return nil
}

// updateTotalBatches update precomputed total_batches based on existing steps and validations.
func (db *PgDB) updateTotalBatches(ctx context.Context, tx *sqlx.Tx, trialID int) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE runs t
		SET total_batches = COALESCE(latest.total_batches_processed, 0)
		FROM (
				SELECT max(m.total_batches) AS total_batches_processed
				FROM metrics m
				WHERE m.trial_id = $1 AND m.archived = false
			) AS latest
		WHERE t.id = $1
		`, trialID); err != nil {
		return errors.Wrap(err, "error computing total_batches")
	}
	return nil
}

func (db *PgDB) _addTrialProfilingMetricsTx(
	ctx context.Context, tx *sqlx.Tx, m *trialv1.TrialMetrics, mGroup model.MetricGroup,
) error {
	if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
		return err
	}

	metrics := model.JSONObj(m.Metrics.AvgMetrics.AsMap())
	_, err := db.addRawMetrics(ctx, tx, &metrics, tryAsTime(m.ReportTime), m.TrialRunId, m.TrialId, nil, mGroup)
	return err
}

func (db *PgDB) _addTrialMetricsTx(
	ctx context.Context, tx *sqlx.Tx, m *trialv1.TrialMetrics, mGroup model.MetricGroup,
) (rollbacks int, err error) {
	isValidation := mGroup == model.ValidationMetricGroup
	mBody := newMetricsBody(m.Metrics.AvgMetrics, m.Metrics.BatchMetrics, isValidation)

	if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
		return rollbacks, err
	}

	if rollbacks, err = rollbackMetrics(ctx, tx, m.TrialRunId, m.TrialId, m.GetStepsCompleted(),
		mGroup); err != nil {
		return rollbacks, err
	}
	var summaryMetrics model.JSONObj
	err = tx.QueryRowContext(ctx, `
		SELECT summary_metrics FROM runs WHERE id = $1 FOR UPDATE;
	`, m.TrialId).Scan(&summaryMetrics)
	if err != nil {
		return rollbacks, fmt.Errorf("error getting summary metrics from trials: %w", err)
	}

	metricRowID, addedMetrics, err := db.addMetricsWithMerge(ctx, tx,
		mBody, tryAsTime(m.ReportTime), m.TrialRunId, m.TrialId, m.StepsCompleted, mGroup)
	if err != nil {
		return rollbacks, err
	}

	switch {
	case rollbacks != 0:
		if err := db.updateTotalBatches(ctx, tx, int(m.TrialId)); err != nil {
			return rollbacks, errors.Wrap(err, "rollback")
		}

		if err := db.updateLatestValidationID(ctx, tx, int(m.TrialId)); err != nil {
			return rollbacks, fmt.Errorf(
				"rollback updating latest validation ID for trial %d: %w", m.TrialId, err)
		}

		if err := db.fullTrialSummaryMetricsRecompute(ctx, tx, int(m.TrialId)); err != nil {
			return rollbacks, errors.Wrap(err, "error on rollback compute of summary metrics")
		}
	default: // no rollbacks happened.
		summaryMetricsJSONPath := model.TrialSummaryMetricsJSONPath(mGroup)
		if _, ok := summaryMetrics[summaryMetricsJSONPath]; !ok {
			summaryMetrics[summaryMetricsJSONPath] = map[string]any{}
		}

		var summaryMetricsForGroup map[string]any
		switch v := summaryMetrics[summaryMetricsJSONPath].(type) {
		case model.JSONObj:
			summaryMetricsForGroup = map[string]any(v)
		case map[string]any:
			summaryMetricsForGroup = v
		default:
			log.Errorf("summary metric "+
				"%+v path %s type %T value %+v is not a map, setting to empty map",
				summaryMetrics,
				summaryMetricsJSONPath,
				summaryMetrics[summaryMetricsJSONPath],
				summaryMetrics[summaryMetricsJSONPath],
			)

			summaryMetricsForGroup = make(map[string]any)
		}
		summaryMetrics[summaryMetricsJSONPath] = calculateNewSummaryMetrics(
			summaryMetricsForGroup,
			addedMetrics.AvgMetrics,
		)

		var latestValidationID *int
		if isValidation {
			var searcherMetric *string
			if err := tx.QueryRowContext(ctx, `
		SELECT experiments.config->'searcher'->>'metric' AS metric_name
		FROM experiments
		JOIN trials t ON t.experiment_id = experiments.id
		WHERE t.id = $1`, int(m.TrialId)).Scan(&searcherMetric); err != nil {
				return rollbacks, fmt.Errorf("getting trial's searcher metric: %w", err)
			}
			if searcherMetric != nil &&
				m.Metrics.AvgMetrics.Fields[*searcherMetric].AsInterface() != nil {
				latestValidationID = &metricRowID
			}
		}

		for k, v := range summaryMetrics {
			switch v := v.(type) {
			case model.JSONObj, map[string]any:
			default:
				log.Errorf("when updating summary metric "+
					"%+v path %s type %T value %+v is not a map, setting to empty map",
					summaryMetrics,
					k,
					v,
					v,
				)
				summaryMetrics[k] = model.JSONObj{}
			}
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE runs SET total_batches = GREATEST(total_batches, $2),
summary_metrics = $3, summary_metrics_timestamp = NOW(),
latest_validation_id = coalesce($4, latest_validation_id)
WHERE id = $1;
`, m.TrialId, m.StepsCompleted, summaryMetrics, latestValidationID); err != nil {
			return rollbacks, errors.Wrap(err, "updating trial total batches")
		}
	}

	if isValidation {
		if err := setTrialBestValidation(
			tx, int(m.TrialId),
			int(m.TrialRunId),
			int(m.GetStepsCompleted())); err != nil {
			return rollbacks, errors.Wrap(err, "updating trial best validation")
		}
	}
	return rollbacks, nil
}

// addTrialMetrics inserts a set of trial metrics to the database.
func (db *PgDB) addTrialMetrics(
	ctx context.Context, m *trialv1.TrialMetrics, mGroup model.MetricGroup,
) (rollbacks int, err error) {
	switch v := m.Metrics.AvgMetrics.Fields["epoch"].AsInterface().(type) {
	case float64, nil:
	default:
		return 0, fmt.Errorf("cannot add metric with non numeric 'epoch' value got %v", v)
	}
	return rollbacks, db.withTransaction(fmt.Sprintf("add trial metrics %s", mGroup),
		func(tx *sqlx.Tx) error {
			switch {
			case slices.Contains(model.ProfilingMetricGroups, mGroup):
				err = db._addTrialProfilingMetricsTx(ctx, tx, m, mGroup)
			default:
				rollbacks, err = db._addTrialMetricsTx(ctx, tx, m, mGroup)
			}
			return err
		})
}

const (
	// InfPostgresString how we store infinity in JSONB in postgres.
	InfPostgresString = "Infinity"
	// NegInfPostgresString how we store -infinity in JSONB in postgres.
	NegInfPostgresString = "-Infinity"
	// NaNPostgresString how we store NaN in JSONB in postgres.
	NaNPostgresString = "NaN"

	// MetricTypeString is the summary metric type for string or mixed types.
	MetricTypeString = "string"
	// MetricTypeNumber is the summary metric type for floats or ints.
	MetricTypeNumber = "number"
	// MetricTypeBool is the summary metric type for boolean.
	MetricTypeBool = "boolean"
	// MetricTypeDate is the summary metric type for date metrics.
	MetricTypeDate = "date"
	// MetricTypeObject is the summary metric type for object types.
	MetricTypeObject = "object"
	// MetricTypeArray is the summary metric type for array types.
	MetricTypeArray = "array"
	// MetricTypeNull is the summary metric type for array types.
	MetricTypeNull = "null"
)

func jsonAnyToFloat(v any) float64 {
	if s, ok := v.(string); ok {
		if f, isSpecial := stringToSpecialFloats(s); isSpecial {
			return f
		}
	}

	if f, ok := v.(float64); ok {
		return f
	}

	if f, ok := v.(int); ok {
		return float64(f)
	}

	log.Errorf("summary metric value expected as float instead got %T %v", v, v)
	return 0.0
}

func stringToSpecialFloats(s string) (float64, bool) {
	switch s {
	case NaNPostgresString:
		return math.NaN(), true
	case InfPostgresString:
		return math.Inf(1), true
	case NegInfPostgresString:
		return math.Inf(-1), true
	default:
		return 0.0, false
	}
}

func replaceSpecialFloatsWithString(v any) any {
	if f, ok := v.(float64); ok {
		switch {
		case math.IsNaN(f):
			return NaNPostgresString
		case math.IsInf(f, 1.0):
			return InfPostgresString
		case math.IsInf(f, -1.0):
			return NegInfPostgresString
		}
	}
	return v
}

var pythonISOFormatRegex = regexp.MustCompile(
	`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$`)

// calculateNewSummaryMetrics calculates new summary metrics from the newly added
// metrics and the existing summary metrics.
func calculateNewSummaryMetrics(
	summaryMetrics model.JSONObj, metrics *structpb.Struct,
) model.JSONObj {
	// Calculate numeric metrics.
	for metricName, metric := range metrics.Fields {
		// Get type of provided metric.
		metricFloatValue := 0.0
		metricType := ""
		switch metricValue := metric.AsInterface().(type) {
		case float64:
			metricFloatValue = metricValue
			metricType = MetricTypeNumber
		case string:
			switch f, ok := stringToSpecialFloats(metricValue); {
			case ok:
				metricFloatValue = f
				metricType = MetricTypeNumber
			case pythonISOFormatRegex.MatchString(metricValue):
				metricType = MetricTypeDate
			default:
				metricType = MetricTypeString
			}
		case bool:
			metricType = MetricTypeBool
		case map[string]any:
			metricType = MetricTypeObject
		case []any:
			metricType = MetricTypeArray
		case nil:
			metricType = MetricTypeNull
		default:
			metricType = MetricTypeString
		}

		// If we haven't seen this metric before just add the type we have.
		var ok bool
		var summaryMetric map[string]any
		if summaryMetric, ok = summaryMetrics[metricName].(map[string]any); !ok {
			summaryMetric = map[string]any{"type": metricType}
		} else if summaryMetric["type"] != metricType {
			// If we have seen this before check if we disagree on types and set to string if we do.
			metricType = "string"
			summaryMetric = map[string]any{"type": metricType}
		}
		summaryMetrics[metricName] = summaryMetric

		if metricType != MetricTypeNumber {
			continue
		}

		// Is this the first time seeing a number metric?
		if _, ok = summaryMetric["count"]; !ok {
			summaryMetric["max"] = replaceSpecialFloatsWithString(metricFloatValue)
			summaryMetric["min"] = replaceSpecialFloatsWithString(metricFloatValue)
			summaryMetric["mean"] = replaceSpecialFloatsWithString(metricFloatValue)
			summaryMetric["sum"] = replaceSpecialFloatsWithString(metricFloatValue)
			summaryMetric["count"] = 1
		} else {
			summaryMetric["min"] = replaceSpecialFloatsWithString(
				math.Min(jsonAnyToFloat(summaryMetric["min"]), metricFloatValue))
			summaryMetric["max"] = replaceSpecialFloatsWithString(
				math.Max(jsonAnyToFloat(summaryMetric["max"]), metricFloatValue))
			summaryMetric["sum"] = replaceSpecialFloatsWithString(
				jsonAnyToFloat(summaryMetric["sum"]) + metricFloatValue)
			// Go parsing odditity treats JSON whole numbers as floats.
			summaryMetric["count"] = int(jsonAnyToFloat(summaryMetric["count"])) + 1
			summaryMetric["mean"] = replaceSpecialFloatsWithString(
				jsonAnyToFloat(summaryMetric["sum"]) / jsonAnyToFloat(summaryMetric["count"]))
		}
	}

	// Add last value for all metrics provided.
	for metricName, sumMetric := range summaryMetrics {
		metric, ok := sumMetric.(map[string]any)
		if !ok {
			// Should not happen.
			log.Errorf("summary metric %T %+v is not a map", sumMetric, sumMetric)
			continue
		}

		metric["last"] = replaceSpecialFloatsWithString(metrics.Fields[metricName])
	}

	return summaryMetrics
}

// AddCheckpointMetadata persists metadata for a completed checkpoint to the database.
func AddCheckpointMetadata(ctx context.Context, m *model.CheckpointV2, runID int) error {
	if m.ReportTime.IsZero() {
		m.ReportTime = time.Now().UTC()
	}
	if m.State == "" {
		m.State = model.CompletedState
	}

	var size int64
	for _, v := range m.Resources {
		size += v
	}
	m.Size = size

	err := Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(m).Exec(ctx); err != nil {
			return fmt.Errorf("inserting checkpoint model: %w", err)
		}

		if _, err := tx.NewInsert().Model(&model.RunCheckpoints{
			RunID:        runID,
			CheckpointID: m.UUID,
		}).Exec(ctx); err != nil {
			return fmt.Errorf("inserting checkpoint run model: %w", err)
		}

		if err := UpdateCheckpointSizeTx(ctx, tx, []uuid.UUID{m.UUID}); err != nil {
			return fmt.Errorf("updating checkpoint size: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error adding checkpoint metadata: %w", err)
	}

	return nil
}

func trialToRunAndTrialV2(
	ctx context.Context, tx bun.IDB, trial *model.Trial,
) (*model.Run, *model.TrialV2, error) {
	var e model.Experiment
	if err := tx.NewSelect().Model(&e).
		Column("project_id").
		Where("id = ?", trial.ExperimentID).
		Scan(ctx, &e); err != nil {
		return nil, nil, fmt.Errorf("getting experiment's project ID %d: %w", trial.ExperimentID, err)
	}

	run, v2 := trial.ToRunAndTrialV2(e.ProjectID)

	return run, v2, nil
}

// checkTrialRunID checks that the trial is currently on the given run.
func checkTrialRunID(ctx context.Context, tx *sqlx.Tx, trialID, runID int32) error {
	var cRunID int
	switch err := tx.QueryRowxContext(ctx, `
SELECT run_id
FROM trials
WHERE id = $1
`, trialID).Scan(&cRunID); {
	case err != nil:
		return errors.Wrap(err, "querying current run")
	case int(runID) != cRunID:
		return api.AsValidationError("invalid run id, %d (reported) != %d (expected)", runID, cRunID)
	default:
		return nil
	}
}

// ValidationByTotalBatches looks up a validation by trial and total batches,
// returning nil if none exists.
func (db *PgDB) ValidationByTotalBatches(trialID, totalBatches int) (*model.TrialMetrics, error) {
	var validation model.TrialMetrics
	// TODO: update to go through `metrics`.
	if err := db.query(`
SELECT id, trial_id, total_batches, end_time, metrics
FROM validations
WHERE trial_id = $1
AND total_batches = $2`, &validation, trialID, totalBatches); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for validation (%v, %v)",
			trialID, totalBatches)
	}
	return &validation, nil
}

// CheckpointByTotalBatches looks up a checkpoint by trial and total batch,
// returning nil if none exists.
func (db *PgDB) CheckpointByTotalBatches(trialID, totalBatches int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT *
FROM checkpoints_view c
WHERE c.trial_id = $1 AND c.steps_completed = $2`, &checkpoint, trialID, totalBatches,
	); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for checkpoint (%v, %v)",
			trialID, totalBatches)
	}
	return &checkpoint, nil
}

// LatestCheckpointForTrial finds the latest completed checkpoint for a trial, returning nil if
// none exists.
func (db *PgDB) LatestCheckpointForTrial(trialID int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT *
FROM checkpoints_view c
WHERE c.trial_id = $1 AND c.state = 'COMPLETED'
ORDER BY c.steps_completed DESC
LIMIT 1`, &checkpoint, trialID); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for latest trial checkpoint (%v)", trialID)
	}
	return &checkpoint, nil
}

// TrialState returns the current state of the given trial.
func (db *PgDB) TrialState(trialID int) (model.State, error) {
	var state model.State
	err := db.sql.QueryRow(`
SELECT state
FROM trials
WHERE id = $1
`, trialID).Scan(&state)
	return state, err
}

// TrialStatus returns the current status of the given trial, including the end time
// without returning all its hparams and other unneeded details. Called in paths hotter
// than TrialByID allows.
func (db *PgDB) TrialStatus(trialID int) (model.State, *time.Time, error) {
	status := struct {
		State   model.State `db:"state"`
		EndTime *time.Time  `db:"end_time"`
	}{}
	err := db.query(`
SELECT state, end_time
FROM trials
WHERE id = $1
`, &status, trialID)
	return status.State, status.EndTime, err
}

// setTrialBestValidation sets `public.trials.best_validation_id` to the `id` of the row in
// `public.validations` corresponding to the trial's best validation.
func setTrialBestValidation(tx *sqlx.Tx, trialID int, trialRunID int, stepsCompleted int) error {
	_, err := tx.Exec(`
WITH const AS (
    SELECT t.id as trial_id,
           config->'searcher'->>'metric' AS metric_name,
           (SELECT
               CASE WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
			   THEN 1
			   ELSE -1 END) AS sign
    FROM experiments e
    INNER JOIN trials t ON t.experiment_id = e.id
  	WHERE t.id = $1
), best_validation AS (
	SELECT
		v.id AS id,
		const.sign * (v.metrics->'validation_metrics'->>const.metric_name)::float8 AS metric,
		(v.metrics->'validation_metrics'->>const.metric_name)::float8 AS searcher_metric_value
	FROM (
		SELECT * FROM validations where id = (select best_validation_id from trials where id = $1)
		UNION ALL
		SELECT * FROM validations
			where trial_id = $1
			and trial_run_id = $2
			and total_batches = $3
	) v, const
	WHERE v.trial_id = $1
	ORDER BY metric ASC
	LIMIT 1
)
UPDATE runs t
SET best_validation_id = (SELECT bv.id FROM best_validation bv),
searcher_metric_value = (SELECT bv.searcher_metric_value FROM best_validation bv),
searcher_metric_value_signed =
(SELECT bv.searcher_metric_value * const.sign FROM best_validation bv, const)
WHERE t.id = $1;
`, trialID, trialRunID, stepsCompleted)
	return errors.Wrapf(err, "error updating best validation for trial %d", trialID)
}

// UpdateCheckpointSizeTx which updates checkpoint size and count to experiment and trial, is duplicated here.
// Remove from this file when bunifying. Original is in master/internal/checkpoints/postgres_checkpoints.go.
func UpdateCheckpointSizeTx(ctx context.Context, idb bun.IDB, checkpoints []uuid.UUID) error {
	if idb == nil {
		idb = Bun()
	}

	var experimentIDs []int
	err := idb.NewRaw(`
UPDATE runs SET checkpoint_size=sub.size, checkpoint_count=sub.count FROM (
	SELECT
		run_id,
		COALESCE(SUM(size) FILTER (WHERE state != 'DELETED'), 0) AS size,
		COUNT(*) FILTER (WHERE state != 'DELETED') AS count
	FROM checkpoints_v2
	JOIN run_checkpoints rc ON rc.checkpoint_id = checkpoints_v2.uuid
	WHERE rc.run_id IN (
		SELECT run_id FROM run_checkpoints WHERE checkpoint_id IN (?)
	)
	GROUP BY run_id
) sub
WHERE runs.id = sub.run_id
RETURNING experiment_id`, bun.In(checkpoints)).Scan(ctx, &experimentIDs)
	if err != nil {
		return errors.Wrap(err, "errors updating trial checkpoint sizes and counts")
	}
	if len(experimentIDs) == 0 { // Checkpoint potentially to non experiment.
		return nil
	}

	uniqueExpIDs := maps.Keys(set.FromSlice(experimentIDs))
	var res bool // Need this since bun.NewRaw() doesn't have a Exec(ctx) method.
	err = idb.NewRaw(`
UPDATE experiments SET checkpoint_size=sub.size, checkpoint_count=sub.count FROM (
	SELECT experiment_id, SUM(checkpoint_size) AS size, SUM(checkpoint_count) as count FROM trials
	WHERE experiment_id IN (?)
	GROUP BY experiment_id
) sub
WHERE experiments.id = sub.experiment_id
RETURNING true`, bun.In(uniqueExpIDs)).Scan(ctx, &res)
	if err != nil {
		return errors.Wrap(err, "errors updating experiment checkpoint sizes and counts")
	}

	return nil
}
