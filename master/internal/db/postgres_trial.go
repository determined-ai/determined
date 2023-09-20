package db

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// AddTrial adds the trial to the database and sets its ID.
func AddTrial(ctx context.Context, trial *model.Trial, taskID model.TaskID) error {
	if trial.ID != 0 {
		return errors.Errorf("error adding a trial with non-zero id %v", trial.ID)
	}

	err := Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(trial).Returning("id").Exec(ctx); err != nil {
			return fmt.Errorf("inserting trial model: %w", err)
		}

		trialTaskID := &model.TrialTaskID{TrialID: trial.ID, TaskID: taskID}
		if _, err := tx.NewInsert().Model(trialTaskID).Exec(ctx); err != nil {
			return fmt.Errorf("inserting trial task id relationship: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("inserting trial %v: %w", trial, err)
	}

	return nil
}

// UpsertTrialByExternalIDTx UPSERTs the trial with respect to the external_trial_id.
func UpsertTrialByExternalIDTx(
	ctx context.Context, tx bun.Tx, trial *model.Trial, taskID model.TaskID,
) error {
	if trial.ID != 0 {
		return errors.Errorf("error adding a trial with non-zero id %v", trial.ID)
	}

	if _, err := tx.NewInsert().Model(trial).
		On("CONFLICT (experiment_id, external_trial_id) DO UPDATE").
		Set("hparams = EXCLUDED.hparams").
		Returning("id").Exec(ctx); err != nil {
		return fmt.Errorf("upserting trial model: %w", err)
	}

	trialTaskID := &model.TrialTaskID{TrialID: trial.ID, TaskID: taskID}
	if _, err := tx.NewInsert().Model(trialTaskID).
		On("CONFLICT (trial_id, task_id) DO NOTHING").Exec(ctx); err != nil {
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

// TrialTaskIDsByTrialID returns trial id task ids by trial ID, sorted by task run ID.
func TrialTaskIDsByTrialID(ctx context.Context, trialID int) ([]*model.TrialTaskID, error) {
	var ids []*model.TrialTaskID
	if err := Bun().NewSelect().Model(&ids).
		Where("trial_id = ?", trialID).
		Join("JOIN tasks t ON trial_task_id.task_id = t.task_id").
		Order("t.start_time").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("getting tasks for trial ID %d: %w", trialID, err)
	}
	return ids, nil
}

// TrialByExperimentAndRequestID looks up a trial, returning an error if none exists.
func TrialByExperimentAndRequestID(
	ctx context.Context, experimentID int, requestID model.RequestID,
) (*model.Trial, error) {
	t := &model.Trial{}
	if err := Bun().NewSelect().Model(t).
		Where("experiment_id = ?", experimentID).
		Where("request_id = ?", requestID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("error querying for trial %s: %w", requestID, err)
	}
	return t, nil
}

// UpdateTrial updates an existing trial. Fields that are nil or zero are not
// updated.  end_time is set if the trial moves to a terminal state.
func (db *PgDB) UpdateTrial(id int, newState model.State) error {
	trial, err := TrialByID(context.TODO(), id)
	if err != nil {
		return errors.Wrapf(err, "error finding trial %v to update", id)
	}

	if trial.State == newState {
		return nil
	}

	if !model.TrialTransitions[trial.State][newState] {
		return errors.Errorf("illegal transition %v -> %v for trial %v",
			trial.State, newState, trial.ID)
	}
	toUpdate := []string{"state"}
	trial.State = newState
	if model.TerminalStates[newState] {
		now := time.Now().UTC()
		trial.EndTime = &now
		toUpdate = append(toUpdate, "end_time")
	}

	return db.withTransaction("update_trial", func(tx *sqlx.Tx) error {
		// Only the trial actor updates this row, and it does so in a serialized
		// fashion already, so this transaction is more a matter of atomicity.
		if err := namedExecOne(tx, fmt.Sprintf(`
UPDATE trials
%v
WHERE id = :id`, setClause(toUpdate)), trial); err != nil {
			return errors.Wrapf(err, "error updating (%v) in trial %v",
				strings.Join(toUpdate, ", "), id)
		}

		if model.TerminalStates[newState] && trial.EndTime != nil {
			return completeTrialsTasks(tx, id, *trial.EndTime)
		}

		return nil
	})
}

// UpdateTrialRunnerState updates a trial runner's state.
func (db *PgDB) UpdateTrialRunnerState(id int, state string) error {
	return db.UpdateTrialRunnerMetadata(id, &trialv1.TrialRunnerMetadata{State: state})
}

// UpdateTrialRunnerMetadata updates a trial's metadata about its runner.
func (db *PgDB) UpdateTrialRunnerMetadata(id int, md *trialv1.TrialRunnerMetadata) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET runner_state = $2
WHERE id = $1`, id, md.State); err != nil {
		return errors.Wrap(err, "saving trial runner state")
	}
	return nil
}

// TrialRunIDAndRestarts returns the run id and restart count for a trial.
func (db *PgDB) TrialRunIDAndRestarts(trialID int) (int, int, error) {
	var runID, restart int
	if err := db.sql.QueryRowx(`
SELECT run_id, restarts
FROM trials
WHERE id = $1`, trialID).Scan(&runID, &restart); err != nil {
		return 0, 0, errors.Wrap(err, "failed to scan trial restart count")
	}
	return runID, restart, nil
}

// UpdateTrialRunID sets the trial's run ID.
func (db *PgDB) UpdateTrialRunID(id, runID int) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET run_id = $2
WHERE id = $1`, id, runID); err != nil {
		return errors.Wrap(err, "updating trial run id")
	}
	return nil
}

// UpdateTrialRestarts sets the trial's restart count.
func (db *PgDB) UpdateTrialRestarts(id, restartCount int) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET restarts = $2
WHERE id = $1`, id, restartCount); err != nil {
		return errors.Wrap(err, "updating trial restarts")
	}
	return nil
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
		if _, ok := v.(map[string]any); !ok {
			log.Errorf("when full compute updating summary metric "+
				"%+v path %s type %T value %+v is not a map, setting to empty map",
				updatedSummaryMetrics,
				k,
				v,
				v,
			)
			updatedSummaryMetrics[k] = make(map[string]any)
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE trials SET summary_metrics = $1,
	summary_metrics_timestamp = NOW() WHERE id = $2`, updatedSummaryMetrics, trialID); err != nil {
		return fmt.Errorf("rollback updating trial summary metrics: %w", err)
	}
	return nil
}

func (db *PgDB) calculateFullTrialSummaryMetrics(
	ctx context.Context, tx *sqlx.Tx, trialID int, metricGroup model.MetricGroup,
) (model.JSONObj, error) {
	partition := customMetricGroupToPartitionType(metricGroup)
	jsonPath := model.TrialMetricsJSONPath(partition == ValidationMetric)
	//nolint: execinquery
	rows, err := tx.QueryContext(ctx, db.queries.GetOrLoad("calculate-full-trial-summary-metrics"),
		trialID, jsonPath, partition, metricGroup)
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
		UPDATE trials SET latest_validation_id = (
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
		UPDATE trials t
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

func (db *PgDB) _addTrialMetricsTx(
	ctx context.Context, tx *sqlx.Tx, m *trialv1.TrialMetrics, mGroup model.MetricGroup,
) (rollbacks int, err error) {
	isValidation := mGroup == model.ValidationMetricGroup
	mBody := newMetricsBody(m.Metrics.AvgMetrics, m.Metrics.BatchMetrics, isValidation)

	if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
		return rollbacks, err
	}

	if rollbacks, err = rollbackMetrics(ctx, tx, m.TrialRunId, m.TrialId, m.StepsCompleted,
		mGroup); err != nil {
		return rollbacks, err
	}
	var summaryMetrics model.JSONObj
	err = tx.QueryRowContext(ctx, `
		SELECT summary_metrics FROM trials WHERE id = $1 FOR UPDATE;
	`, m.TrialId).Scan(&summaryMetrics)
	if err != nil {
		return rollbacks, fmt.Errorf("error getting summary metrics from trials: %w", err)
	}

	metricRowID, addedMetrics, err := db.addMetricsWithMerge(ctx, tx,
		mBody, m.TrialRunId, m.TrialId, m.StepsCompleted, mGroup)
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
		if g, ok := summaryMetrics[summaryMetricsJSONPath].(map[string]any); ok {
			summaryMetricsForGroup = g
		} else {
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
			if _, ok := v.(map[string]any); !ok {
				log.Errorf("when updating summary metric "+
					"%+v path %s type %T value %+v is not a map, setting to empty map",
					summaryMetrics,
					k,
					v,
					v,
				)
				summaryMetrics[k] = make(map[string]any)
			}
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE trials SET total_batches = GREATEST(total_batches, $2),
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
			int(m.StepsCompleted)); err != nil {
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
			rollbacks, err = db._addTrialMetricsTx(ctx, tx, m, mGroup)
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
func AddCheckpointMetadata(ctx context.Context, m *model.CheckpointV2) error {
	var size int64
	for _, v := range m.Resources {
		size += v
	}
	m.Size = size

	err := Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(m).Exec(context.TODO()); err != nil {
			return errors.Wrap(err, "inserting checkpoint")
		}

		if err := UpdateCheckpointSizeTx(ctx, tx, []uuid.UUID{m.UUID}); err != nil {
			return errors.Wrap(err, "updating checkpoint size")
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error adding checkpoint metadata: %w", err)
	}

	return nil
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
UPDATE trials t
SET best_validation_id = (SELECT bv.id FROM best_validation bv),
searcher_metric_value = (SELECT bv.searcher_metric_value FROM best_validation bv),
searcher_metric_value_signed =
(SELECT bv.searcher_metric_value * const.sign FROM best_validation bv, const)
WHERE t.id = $1;
`, trialID, trialRunID, stepsCompleted)
	return errors.Wrapf(err, "error updating best validation for trial %d", trialID)
}
