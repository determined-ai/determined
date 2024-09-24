package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/db/bunutils"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

const (
	asc  = "" // This is blank because ascending is the default
	desc = "DESC"
	max  = "max"
	min  = "min"
)

// ProjectExperiments returns a list of experiments within a project.
func ProjectExperiments(ctx context.Context, pID int) (experiments []*model.Experiment, err error) {
	rows, err := Bun().NewSelect().
		Model((*model.Experiment)(nil)).
		Column("experiment.id").
		Column("state").
		Column("config").
		Column("start_time").
		Column("end_time").
		Column("archived").
		Column("owner_id").
		Column("notes").
		Column("job_id").
		Column("project_id").
		Column("unmanaged").
		ColumnExpr("u.username as username").
		Join("JOIN users AS u ON (experiment.owner_id = u.id)").
		Where("experiment.project_id = ?", pID).
		Rows(ctx)
	if err != nil {
		return nil, fmt.Errorf("selecting project experiments: %w", err)
	}

	defer rows.Close()
	for rows.Next() {
		var exp model.Experiment
		if err := Bun().ScanRow(ctx, rows, &exp); err != nil {
			return nil, fmt.Errorf("reading experiment from db: %w", err)
		}
		experiments = append(experiments, &exp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("selecting project experiments: %w", err)
	}

	return experiments, nil
}

// ExperimentLabelUsage returns a flattened and deduplicated list of all the
// labels in use across all experiments.
func (db *PgDB) ExperimentLabelUsage(projectID int32) (labelUsage map[string]int, err error) {
	// First, assemble all the JSON lists that the database returns into a
	// single tally of all the labels
	type dbLabelList struct {
		Labels []byte
	}
	var rawLists []dbLabelList
	err = db.Query("get_experiment_labels", &rawLists, projectID)
	if err != nil {
		return nil, fmt.Errorf("error in get_experiment_labels query: %w", err)
	}
	labelUsage = make(map[string]int)
	for _, rawList := range rawLists {
		if len(rawList.Labels) == 0 {
			continue
		}
		var parsedList []string
		err = json.Unmarshal(rawList.Labels, &parsedList)
		if err != nil {
			return nil, fmt.Errorf("error parsing experiment labels: %w", err)
		}
		for i := range parsedList {
			label := parsedList[i]
			labelUsage[label]++
		}
	}
	return labelUsage, nil
}

// GetExperimentStatus returns the current state of the experiment.
func (db *PgDB) GetExperimentStatus(experimentID int) (state model.State, progress float64,
	err error,
) {
	row := db.sql.QueryRow(
		"SELECT state, COALESCE(progress, 0) as progress FROM experiments WHERE id=$1",
		experimentID)
	err = row.Scan(&state, &progress)
	return state, progress, err
}

// GetNonTerminalExperimentCount returns the number of non terminal experiments.
func GetNonTerminalExperimentCount(ctx context.Context,
	experimentIDs []int32,
) (count int, err error) {
	c, err := Bun().NewSelect().Table("experiments").
		Where("id IN (?)", bun.In(experimentIDs)).
		Where("state NOT IN (?)", bun.In(model.StatesToStrings(model.TerminalStates))).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("counting non terminal experiments IDs %v: %w", experimentIDs, err)
	}

	return c, nil
}

// MetricNames returns a list of metric names for the given experiment IDs.
func (db *PgDB) MetricNames(ctx context.Context, experimentIDs []int) (
	map[model.MetricGroup][]string, error,
) {
	type MetricNamesRow struct {
		MetricName string
		JSONPath   string
	}
	rows := []MetricNamesRow{}

	metricNames := BunSelectMetricGroupNames().Distinct().
		Where("experiment_id IN (?)", bun.In(experimentIDs))

	err := Bun().NewSelect().TableExpr("(?) as metric_names", metricNames).
		Column("json_path").
		Column("metric_name").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	metricNamesMap := make(map[model.MetricGroup][]string)
	for _, row := range rows {
		mGroup := model.TrialSummaryMetricGroup(row.JSONPath)
		if _, ok := metricNamesMap[mGroup]; !ok {
			metricNamesMap[mGroup] = make([]string, 0)
		}
		metricNamesMap[mGroup] = append(metricNamesMap[mGroup], row.MetricName)
	}

	return metricNamesMap, nil
}

type batchesWrapper struct {
	Batches int32     `db:"batches_processed" bun:"batches_processed"`
	EndTime time.Time `db:"end_time"`
}

// MetricBatches returns the milestones (in batches processed) at which a specific metric
// was recorded.
func MetricBatches(
	experimentID int, metricName string, startTime time.Time, metricGroup model.MetricGroup,
) (
	batches []int32, endTime time.Time, err error,
) {
	var rows []*batchesWrapper
	jsonKey := model.TrialMetricsJSONPath(metricGroup == model.ValidationMetricGroup)

	err = BunSelectMetricsQuery(metricGroup, false).
		TableExpr("trials t").
		Join("INNER JOIN metrics m ON t.id=m.trial_id").
		ColumnExpr("m.total_batches AS batches_processed, max(t.end_time) as end_time").
		Where("t.experiment_id = ?", experimentID).
		Where(fmt.Sprintf("m.metrics->'%s' ? '%s'", jsonKey, metricName)).
		Where("m.end_time > ?", startTime).
		Group("batches_processed").Scan(context.Background(), &rows)
	if err != nil {
		return nil, endTime, errors.Wrapf(err, "error querying DB for metric batches")
	}
	for _, row := range rows {
		batches = append(batches, row.Batches)
		if row.EndTime.After(endTime) {
			endTime = row.EndTime
		}
	}

	return batches, endTime, nil
}

// TrainingMetricBatches returns the milestones (in batches processed) at which a specific training
// metric was recorded.
func (db *PgDB) TrainingMetricBatches(experimentID int, metricName string, startTime time.Time) (
	batches []int32, endTime time.Time, err error,
) {
	return MetricBatches(experimentID, metricName, startTime, model.TrainingMetricGroup)
}

// ValidationMetricBatches returns the milestones (in batches processed) at which a specific
// validation metric was recorded.
func (db *PgDB) ValidationMetricBatches(experimentID int, metricName string, startTime time.Time) (
	batches []int32, endTime time.Time, err error,
) {
	return MetricBatches(experimentID, metricName, startTime, model.ValidationMetricGroup)
}

type snapshotWrapper struct {
	TrialID int32     `db:"trial_id"`
	Hparams []byte    `db:"hparams"`
	Metric  float64   `db:"metric"`
	EndTime time.Time `db:"end_time"`
	Batches int32     `db:"batches"`
}

func snapshotWrapperToTrial(r snapshotWrapper) (*apiv1.TrialsSnapshotResponse_Trial, error) {
	var trial apiv1.TrialsSnapshotResponse_Trial
	trial.TrialId = r.TrialID

	var inter map[string]interface{}
	err := json.Unmarshal(r.Hparams, &inter)
	if err != nil {
		return nil, err
	}
	trial.Hparams = protoutils.ToStruct(inter)
	trial.Metric = r.Metric
	trial.BatchesProcessed = r.Batches
	return &trial, nil
}

// TrialsSnapshot returns metrics across each trial in an experiment at a
// specific point of progress, for metric groups other than training and validation.
func (db *PgDB) TrialsSnapshot(experimentID int, minBatches int, maxBatches int,
	metricName string, startTime time.Time, metricGroup model.MetricGroup,
) (trials []*apiv1.TrialsSnapshotResponse_Trial, endTime time.Time, err error) {
	var rows []snapshotWrapper

	metricPath := model.TrialMetricsJSONPath(metricGroup == model.ValidationMetricGroup)
	mGroupString := string(metricGroup)
	pType := customMetricGroupToPartitionType(&mGroupString)

	err = db.queryRows(`
SELECT
  t.id AS trial_id,
  t.hparams AS hparams,
  (s.metrics->'`+metricPath+`'->>$1)::float8 AS metric,
  s.end_time AS end_time,
  s.total_batches as batches
FROM trials t
	INNER JOIN metrics s ON t.id=s.trial_id
WHERE t.experiment_id=$2
  AND s.total_batches>=$3
  AND s.total_batches<=$4
  AND s.metrics->'`+metricPath+`'->$1 IS NOT NULL
  AND s.end_time > $5
	AND s.metric_group = $6
	AND partition_type = $7
ORDER BY s.end_time;`, &rows, metricName, experimentID, minBatches, maxBatches, startTime, metricGroup, pType)
	if err != nil {
		return nil, endTime, errors.Wrapf(err,
			"failed to get snapshot for experiment %d and generic metric %s.%s",
			experimentID, metricGroup, metricName)
	}
	for _, row := range rows {
		trial, err := snapshotWrapperToTrial(row)
		if err != nil {
			return nil, endTime, errors.Wrap(err, "Failed to process trial metadata")
		}
		trials = append(trials, trial)
		if row.EndTime.After(endTime) {
			endTime = row.EndTime
		}
	}

	return trials, endTime, nil
}

// TopTrialsByMetric chooses the subset of trials from an experiment that recorded the best values
// for the specified metric at any point during the trial.
func TopTrialsByMetric(
	ctx context.Context, experimentID int, maxTrials int, metric string, smallerIsBetter bool,
) ([]int32, error) {
	query := Bun().NewSelect().Table("trials").
		Column("id").
		ColumnExpr("summary_metrics->'validation_metrics'->? AS summary_metrics", metric).
		Where("experiment_id = ?", experimentID).
		Limit(maxTrials)
	if smallerIsBetter {
		query = query.OrderExpr(
			"(summary_metrics->'validation_metrics'->?->>'min')::float ASC NULLS LAST", metric)
	} else {
		query = query.OrderExpr(
			"(summary_metrics->'validation_metrics'->?->>'max')::float DESC NULLS LAST", metric)
	}

	var res []struct {
		ID             int
		SummaryMetrics *map[string]any
	}
	if err := query.Scan(ctx, &res); err != nil {
		return nil, errors.Wrapf(err,
			"error getting top trials for metric for experiment ID %d", experimentID)
	}

	// Return an error if any result was non numeric.
	// This is somewhat weird behavior given we don't return an error for nulls
	// but doing this to keep compatibility with old query.
	trials := make([]int32, 0, len(res))
	for _, r := range res {
		if r.SummaryMetrics != nil && (*r.SummaryMetrics)["count"] == nil {
			return nil, fmt.Errorf("error getting top trials for experimentID %d and metric %s "+
				"because trial %d has reported a non numeric value for this report",
				experimentID, metric, r.ID)
		}

		trials = append(trials, int32(r.ID))
	}

	return trials, nil
}

// TopTrialsByTrainingLength chooses the subset of trials that has been training for the highest
// number of batches, using the specified metric as a tie breaker.
func (db *PgDB) TopTrialsByTrainingLength(experimentID int, maxTrials int, metric string,
	smallerIsBetter bool,
) (trials []int32, err error) {
	order := desc
	aggregate := max
	if smallerIsBetter {
		order = asc
		aggregate = min
	}

	err = db.sql.Select(&trials, fmt.Sprintf(`
SELECT t.id FROM (
  SELECT t.id,
    max(v.total_batches) as progress,
    %s((v.metrics->'validation_metrics'->>$1)::float8) as best_metric
  FROM trials t
  JOIN validations v ON t.id = v.trial_id
  WHERE t.experiment_id=$2
  GROUP BY t.id
  ORDER BY progress DESC, best_metric %s
  LIMIT $3
) t;`, aggregate, order), metric, experimentID, maxTrials)
	return trials, err
}

// MetricMeasurements represents a metric measured by all possible
// independent variables.
type MetricMeasurements struct {
	Values  map[string]interface{}
	Batches uint
	Time    time.Time
	Epoch   *float64 `json:"epoch,omitempty"`
	TrialID int32
}

// ExperimentBestSearcherValidation returns the best searcher validation for an experiment.
func ExperimentBestSearcherValidation(ctx context.Context, id int) (float32, error) {
	exp, err := ExperimentByID(ctx, id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get experiment config")
	}

	metricOrdering := desc
	if exp.Config.Searcher.SmallerIsBetter {
		metricOrdering = asc
	}

	var metric float32
	if err := Bun().NewRaw(fmt.Sprintf(`
SELECT (v.metrics->'validation_metrics'->>?)::float8 as metric
FROM validations v, trials t
WHERE v.trial_id = t.id
  AND t.experiment_id = ?
ORDER BY metric %s
LIMIT 1`, metricOrdering), exp.Config.Searcher.Metric, id).Scan(ctx, &metric); err != nil {
		return 0, MatchSentinelError(err)
	}
	return metric, nil
}

// ExperimentConfigRaw returns the full config object for an experiment as a JSON string.
func (db *PgDB) ExperimentConfigRaw(id int) ([]byte, error) {
	return db.rawQuery(`
SELECT config
FROM experiments
WHERE id = $1`, id)
}

// AddExperiment adds the experiment to the database and sets its ID.
//
// TODO(ilia): deprecate and use module function instead.
func (db *PgDB) AddExperiment(
	experiment *model.Experiment, modelDef []byte, activeConfig expconf.ExperimentConfig,
) (err error) {
	return AddExperiment(context.TODO(), experiment, modelDef, activeConfig)
}

// AddExperiment adds the experiment to the database and sets its ID.
func AddExperiment(
	ctx context.Context,
	experiment *model.Experiment,
	modelDef []byte,
	activeConfig expconf.ExperimentConfig,
) (err error) {
	return Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return AddExperimentTx(ctx, tx, experiment, modelDef, activeConfig, false)
	})
}

// AddExperimentTx adds the experiment to the database and sets its ID.
func AddExperimentTx(
	ctx context.Context, idb bun.IDB,
	experiment *model.Experiment,
	modelDef []byte,
	activeConfig expconf.ExperimentConfig,
	upsert bool,
) (err error) {
	if experiment.ID != 0 {
		return errors.Errorf("error adding an experiment with non-zero id %v", experiment.ID)
	}

	activeConfigStr, err := json.Marshal(activeConfig)
	if err != nil {
		return errors.Wrapf(err, "error handling experiment config %v", activeConfig)
	}

	job := model.Job{
		JobID:   experiment.JobID,
		JobType: model.JobTypeExperiment,
		OwnerID: experiment.OwnerID,
	}
	if _, err = idb.NewInsert().Model(&job).Exec(ctx); err != nil {
		return errors.Wrapf(err, "error inserting job %v", job)
	}

	q := idb.NewInsert().Model(experiment).
		ExcludeColumn("id", "username").
		Value("progress", "?", 0).
		Value("config", "?", string(activeConfigStr)).
		Value("model_definition", "?", modelDef).
		Returning("id")

	if upsert {
		// TODO(ilia): are there any fields user will expect us to update here?
		// `config` field will cover the metadata: name, data, description, labels, etc.
		// No-op SET for external_experiment_id is required for `RETURNING` clause to work.
		q = q.On("CONFLICT (external_experiment_id) DO UPDATE").
			Set("external_experiment_id = EXCLUDED.external_experiment_id").
			Set("config = EXCLUDED.config")
		// TODO(ilia): do something with the job we've already created.
		// Option A) make jobs nullable/optional for unmanaged experiments.
		// Option B) just delete it.
	}

	_, err = q.Exec(ctx)
	if err != nil {
		return errors.Wrapf(err, "error inserting experiment %v", experiment)
	}
	return nil
}

// ExperimentByID looks up an experiment by ID in a database, returning an error if none exists.
func ExperimentByID(ctx context.Context, expID int) (*model.Experiment, error) {
	var experiment model.Experiment

	if err := Bun().NewRaw(`
SELECT e.id, state, config, start_time, end_time, archived,
	   owner_id, notes, job_id, u.username as username, project_id, unmanaged, external_experiment_id
FROM experiments e
JOIN users u ON (e.owner_id = u.id)
WHERE e.id = ?`, expID).Scan(ctx, &experiment); err != nil {
		return nil, MatchSentinelError(err)
	}

	return &experiment, nil
}

// ExperimentByTrialID looks up an experiment by a given trialID, returning an error
// if none exists.
func ExperimentByTrialID(ctx context.Context, trialID int) (*model.Experiment, error) {
	var experiment model.Experiment

	if err := Bun().NewRaw(`
SELECT e.id, e.state, e.config, e.start_time, e.end_time, e.archived,
       e.owner_id, e.notes, e.job_id, u.username as username, e.project_id, unmanaged, external_experiment_id
FROM experiments e
JOIN trials t ON e.id = t.experiment_id
JOIN users u ON (e.owner_id = u.id)
WHERE t.id = ?`, trialID).Scan(ctx, &experiment); err != nil {
		return nil, MatchSentinelError(err)
	}

	return &experiment, nil
}

// ExperimentsByTrialID looks up an experiment by a given list of trialIDs, returning
// an error if none exists.
func ExperimentsByTrialID(ctx context.Context, trialIDs []int) ([]*model.Experiment, error) {
	var experiment []*model.Experiment

	if err := Bun().NewRaw(`
SELECT DISTINCT e.id, e.state, e.config, e.start_time, e.end_time, e.archived,
       e.owner_id, e.notes, e.job_id, u.username as username, e.project_id, unmanaged, external_experiment_id
FROM experiments e
JOIN trials t ON e.id = t.experiment_id
JOIN users u ON (e.owner_id = u.id)
WHERE t.id IN (?)`, bun.In(trialIDs)).Scan(ctx, &experiment); err != nil {
		return nil, MatchSentinelError(err)
	}

	if len(experiment) == 0 {
		return nil, ErrNotFound
	}
	return experiment, nil
}

// ExperimentByTaskID looks up an experiment by a given taskID, returning an error
// if none exists.
func ExperimentByTaskID(
	ctx context.Context, taskID model.TaskID,
) (*model.Experiment, error) {
	var experiment model.Experiment
	if err := Bun().NewRaw(`
SELECT e.id, e.state, e.config, e.start_time, e.end_time, e.archived,
       e.owner_id, e.notes, e.job_id, u.username as username, e.project_id, e.unmanaged, external_experiment_id
FROM experiments e
JOIN trials t ON e.id = t.experiment_id
JOIN run_id_task_id ON t.id = run_id_task_id.run_id
JOIN users u ON e.owner_id = u.id
WHERE run_id_task_id.task_id = ?`, taskID).Scan(ctx, &experiment); err != nil {
		return nil, MatchSentinelError(err)
	}

	return &experiment, nil
}

// ExperimentByExternalIDTx looks up an experiment by a given external experiment id.
func ExperimentByExternalIDTx(ctx context.Context, idb bun.IDB, externalExperimentID string) (
	*model.Experiment, error,
) {
	var experiment model.Experiment

	if err := idb.NewRaw(`
	SELECT e.id, state, config, start_time, end_time, archived,owner_id, notes,
		job_id, u.username as username, project_id, unmanaged, external_experiment_id
	FROM experiments e
	JOIN users u ON (e.owner_id = u.id)
	WHERE e.external_experiment_id = ?`, externalExperimentID).Scan(ctx, &experiment); err != nil {
		return nil, MatchSentinelError(err)
	}

	return &experiment, nil
}

// LegacyExperimentConfigByID parses very old configs, returning a LegacyConfig which
// exposes a select subset of fields in a type-safe way.
func (db *PgDB) LegacyExperimentConfigByID(
	id int,
) (expconf.LegacyConfig, error) {
	var byts []byte
	if err := db.sql.QueryRow(
		"SELECT config FROM experiments WHERE id = $1", id).Scan(&byts); err != nil {
		return expconf.LegacyConfig{}, errors.Wrap(err, "querying legacy config bytes")
	}

	config, err := expconf.ParseLegacyConfigJSON(byts)
	if err != nil {
		return expconf.LegacyConfig{}, errors.Wrap(err, "parsing legacy conf from database")
	}

	return config, nil
}

// ExperimentIDByTrialID looks up an experiment ID by a trial ID.
func (db *PgDB) ExperimentIDByTrialID(trialID int) (int, error) {
	var experimentID int
	if err := db.sql.Get(&experimentID, `
SELECT experiment_id FROM trials where id = $1
`, trialID); err != nil {
		return 0, errors.Wrapf(err, "querying for experiment id for trial %v", trialID)
	}
	return experimentID, nil
}

// NonTerminalExperiments finds all experiments in the database whose states are not terminal.
func (db *PgDB) NonTerminalExperiments() ([]*model.Experiment, error) {
	rows, err := db.sql.Queryx(`
SELECT e.id, state, config, start_time, end_time, archived, owner_id, job_id,
       u.username as username, project_id, unmanaged
FROM experiments e
JOIN users u ON e.owner_id = u.id
WHERE unmanaged = false AND state IN (
	'ACTIVE', 'PAUSED', 'STOPPING_CANCELED', 'STOPPING_COMPLETED', 'STOPPING_ERROR',
	'STOPPING_KILLED'
)`)
	if err == sql.ErrNoRows {
		return nil, errors.WithStack(ErrNotFound)
	} else if err != nil {
		return nil, errors.Wrap(err, "querying for active experiments")
	}

	defer rows.Close()

	var exps []*model.Experiment
	for rows.Next() {
		var exp model.Experiment
		if err := rows.StructScan(&exp); err != nil {
			// Log this error, but try to figure out the experiment ID first.
			configErr := err

			items, err := rows.SliceScan()
			if err != nil {
				log.WithError(configErr).Errorf("failed to read non-terminal experiment config")
				return nil, errors.Wrap(err, "unable to read experiment from db")
			}

			expID, ok := items[0].(int64)
			if !ok {
				log.WithError(configErr).Errorf("failed to read non-terminal experiment config")
				return nil, errors.Errorf(
					"Expected an integer experiment ID, but got: %s", reflect.TypeOf(items[0]))
			}

			log.WithError(configErr).Errorf(
				"failed to read non-terminal experiment config for experiment %v", expID,
			)

			err = db.TerminateExperimentInRestart(int(expID), model.ErrorState)
			if err != nil {
				log.WithError(err).Error("failed to mark experiment as errored")
			}
			continue
		}

		exps = append(exps, &exp)
	}
	return exps, nil
}

// FailDeletingExperiment finds all experiments that were deleting when the master crashed and moves
// them to DELETE_FAILED.
func (db *PgDB) FailDeletingExperiment() error {
	if _, err := db.sql.Exec(`
UPDATE experiments
SET state = 'DELETE_FAILED'
WHERE state = 'DELETING'
`); err != nil {
		return errors.Wrap(err, "failing deleting experiments")
	}
	return nil
}

// TerminateExperimentInRestart is used during master restart to properly terminate an experiment
// which was either in the process of stopping or which is not restorable for some reason, such as
// an invalid experiment config after a version upgrade.
func (db *PgDB) TerminateExperimentInRestart(id int, state model.State) error {
	if _, ok := model.TerminalStates[state]; !ok {
		return errors.Errorf("state %v is not a terminal state", state)
	}

	now := time.Now().UTC()

	tx, err := db.sql.Begin()
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}
	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("during rollback: %v", rErr)
		}
	}()

	// Terminate trials.
	if _, err = tx.Exec(
		`UPDATE runs SET state=$1, end_time=$2 WHERE experiment_id=$3 and end_time IS NULL`,
		state,
		now,
		id,
	); err != nil {
		return errors.Wrap(err, "terminating trials of a stopping experiment")
	}

	// Terminate experiment.
	if _, err = tx.Exec(
		`UPDATE experiments SET state=$1, end_time=$2, progress=NULL WHERE id=$3`, state, now, id,
	); err != nil {
		return errors.Wrap(err, "terminating a stopping experiment")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrapf(err, "committing termination of stopping experiment %v", id)
	}

	tx = nil

	return nil
}

// SaveExperimentConfig saves the current experiment config to the database.
func (db *PgDB) SaveExperimentConfig(id int, config expconf.ExperimentConfig) error {
	query := `
UPDATE experiments
SET config=$1
WHERE id = $2`
	_, err := db.sql.Exec(query, config, id)
	return err
}

// SaveExperimentState saves the current experiment state to the database.
func (db *PgDB) SaveExperimentState(experiment *model.Experiment) error {
	query := `
UPDATE experiments
SET state=:state, end_time=:end_time
WHERE id = :id`
	return db.namedExecOne(query, experiment)
}

// TrySaveExperimentState saves the current experiment state to the database and
// returns if we successfully changed the state or not.
func (db *PgDB) TrySaveExperimentState(experiment *model.Experiment) error {
	var newState, oldState model.State
	if err := db.sql.QueryRowx(`
UPDATE experiments e
SET state=$2
FROM (SELECT state FROM experiments WHERE id = $1 FOR UPDATE) old
WHERE e.id = $1
RETURNING e.state, old.state
`, experiment.ID, experiment.State).Scan(&newState, &oldState); err != nil {
		return errors.Wrap(err, "updating experiment state")
	}
	if newState == oldState {
		return fmt.Errorf("could not transition experiment")
	}
	return nil
}

// SaveExperimentArchiveStatus saves the current experiment archive status to the database.
func (db *PgDB) SaveExperimentArchiveStatus(experiment *model.Experiment) error {
	if !model.TerminalStates[experiment.State] {
		return errors.Errorf("cannot set archived for experiment in state %v", experiment.State)
	}

	query := `
UPDATE experiments
SET archived=:archived
WHERE id = :id`
	return db.namedExecOne(query, experiment)
}

// DeleteExperiments deletes zero or more experiments.
func (db *PgDB) DeleteExperiments(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	err := Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model(&model.CheckpointV2{}).
			Where(`task_id IN (
	SELECT tt.task_id
	FROM run_id_task_id tt
	JOIN trials t ON t.id = tt.run_id
	WHERE experiment_id IN (?)
)`, bun.In(ids)).
			Exec(ctx); err != nil {
			return fmt.Errorf("deleting checkpoints (v2): %w", err)
		}

		if _, err := tx.NewDelete().Model(&model.Experiment{}).
			Where("id IN (?)", bun.In(ids)).
			Exec(ctx); err != nil {
			return fmt.Errorf("deleting from experiments table: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("deleting experiments %v: %w", ids, err)
	}

	return nil
}

// ExperimentHasCheckpointsInRegistry checks if the experiment has any checkpoints in the registry.
func (db *PgDB) ExperimentHasCheckpointsInRegistry(id int) (bool, error) {
	var exists bool
	err := db.sql.QueryRow(`
SELECT
EXISTS(
   SELECT 1
   FROM experiments e
   JOIN checkpoints_view c ON c.experiment_id = e.id
   JOIN model_versions mv ON mv.checkpoint_uuid = c.uuid
   WHERE e.id = $1
)`, id).Scan(&exists)
	return exists, err
}

// SaveExperimentProgress stores the progress for an experiment in the database.
func (db *PgDB) SaveExperimentProgress(id int, progress *float64) error {
	if progress != nil && (*progress < 0 || *progress > 1) {
		return errors.Errorf("invalid progress value: %f. Progress value should be between 0 and 1", *progress)
	}
	res, err := db.sql.Exec(`UPDATE experiments SET progress = $1 WHERE id = $2`, progress, id)
	if err != nil {
		return errors.Wrap(err, "saving experiment progress")
	}
	if numRows, err := res.RowsAffected(); err != nil {
		return errors.Wrap(err, "checking affected rows for saving experiment progress")
	} else if numRows != 1 {
		return errors.Errorf("saving experiment %d's progress affected %d rows instead of 1", id, numRows)
	}
	return nil
}

// ActiveExperimentConfig returns the full config object for an experiment.
func (db *PgDB) ActiveExperimentConfig(id int) (expconf.ExperimentConfig, error) {
	expConfigBytes, err := db.rawQuery(`
SELECT config
FROM experiments
WHERE id = $1`, id)
	if err != nil {
		return expconf.ExperimentConfig{}, err
	}
	expConfig, err := expconf.ParseAnyExperimentConfigYAML(expConfigBytes)
	if err != nil {
		return expconf.ExperimentConfig{}, errors.WithStack(err)
	}
	return schemas.WithDefaults(expConfig), nil
}

// ActiveLogPolicies returns log pattern policies for an experiment ID.
// This should only be called on a running experiment.
func ActiveLogPolicies(
	ctx context.Context, id int,
) (expconf.LogPoliciesConfig, error) {
	res := struct {
		LogPolicies expconf.LogPoliciesConfig
	}{}
	if err := Bun().NewSelect().Table("experiments").
		ColumnExpr("config -> 'log_policies' AS log_policies").
		Where("id = ?", id).
		Scan(ctx, &res); err != nil {
		return nil, fmt.Errorf("getting log pattern policies config for experiment %d: %w", id, err)
	}

	return res.LogPolicies, nil
}

// ExperimentTotalStepTime returns the total elapsed time for all allocations of the experiment
// with the given ID. Any step with a NULL end_time does not contribute. Elapsed time is
// expressed as a floating point number of seconds.
func ExperimentTotalStepTime(ctx context.Context, id int) (float64, error) {
	var seconds float64
	if err := Bun().NewSelect().
		ColumnExpr("COALESCE(extract(epoch from sum(a.end_time - a.start_time)), 0)").
		TableExpr("allocations AS a").
		Join("JOIN run_id_task_id AS tasks ON a.task_id = tasks.task_id").
		Join("JOIN trials AS t ON tasks.run_id = t.id").
		Where("t.experiment_id = ?", id).
		Scan(ctx, &seconds); err != nil {
		return 0.0, fmt.Errorf("querying for total step time of experiment %v: %w", id, err)
	}
	return seconds, nil
}

// ExperimentNumTrials returns the total number of trials for the experiment.
func (db *PgDB) ExperimentNumTrials(id int) (int64, error) {
	var numTrials int64
	if err := db.sql.Get(&numTrials, `
SELECT count(*)
FROM trials
WHERE trials.experiment_id = $1
`, id); err != nil {
		return 0, errors.Wrapf(err, "querying for number of trials of experiment %v", id)
	}
	return numTrials, nil
}

// ExperimentTrialIDs returns the trial IDs for the experiment.
func (db *PgDB) ExperimentTrialIDs(expID int) ([]int, error) {
	var trialIDRows []struct {
		ID int
	}
	if err := db.queryRows(`
SELECT id
FROM trials
WHERE trials.experiment_id = $1
`, &trialIDRows, expID); err != nil {
		return nil, errors.Wrapf(err, "querying for trial IDs of experiment %v", expID)
	}
	var trialIDs []int
	for _, r := range trialIDRows {
		trialIDs = append(trialIDs, r.ID)
	}
	return trialIDs, nil
}

// ExperimentsTrialAndTaskIDs returns the trial and task IDs for one or more experiments.
func ExperimentsTrialAndTaskIDs(ctx context.Context, idb bun.IDB, expIDs []int) (
	[]int, []model.TaskID, error,
) {
	if len(expIDs) == 0 {
		return nil, nil, nil
	}

	var res []model.RunTaskID
	if err := idb.NewSelect().Model(&res).
		Join("JOIN trials ON trials.id = run_task_id.run_id").
		Where("trials.experiment_id IN (?)", bun.In(expIDs)).
		Scan(ctx); err != nil {
		return nil, nil, fmt.Errorf("querying for trial / task IDs of experiments %v: %w", expIDs, err)
	}

	var taskIDs []model.TaskID
	trialIDsMap := make(map[int]bool)
	for _, r := range res {
		trialIDsMap[r.RunID] = true
		taskIDs = append(taskIDs, r.TaskID)
	}

	return maps.Keys(trialIDsMap), taskIDs, nil
}

// ExperimentNumSteps returns the total number of steps for all trials of the experiment.
func ExperimentNumSteps(ctx context.Context, id int) (int64, error) {
	numSteps, err := Bun().NewSelect().
		TableExpr("raw_steps AS s").
		Join("JOIN trials AS t ON t.id = s.trial_id").
		Where("t.experiment_id = ?", id).
		Count(ctx)
	if err != nil {
		return int64(0), fmt.Errorf("querying for number of steps of experiment %v: %w", id, err)
	}

	return int64(numSteps), nil
}

// ExperimentModelDefinitionRaw returns the zipped model definition for an experiment as a byte
// array.
func (db *PgDB) ExperimentModelDefinitionRaw(id int) ([]byte, error) {
	return db.rawQuery(`
SELECT model_definition
FROM experiments
WHERE id = $1`, id)
}

// GetCheckpoint gets checkpointv1.Checkpoint from the database by UUID.
// Can be moved to master/internal/checkpoints once db/postgres_model_intg_test is bunified.
// WARNING: Function does not account for "NaN", "Infinity", or "-Infinity" due to Bun unmarshallling.
func GetCheckpoint(ctx context.Context, checkpointUUID string) (*checkpointv1.Checkpoint, error) {
	var retCkpt1 checkpointv1.Checkpoint
	err := Bun().NewSelect().
		TableExpr("proto_checkpoints_view").
		ColumnExpr("proto_time(report_time) as report_time").
		Column("task_id").
		Column("allocation_id").
		Column("uuid").
		Column("resources").
		Column("metadata").
		ColumnExpr(bunutils.ProtoStateDBCaseString(checkpointv1.State_value, "state", "state",
			"")).
		Column("training").
		Column("storage_id").
		Where("uuid = ?::uuid", checkpointUUID).Scan(ctx, &retCkpt1)
	if err != nil {
		return nil, fmt.Errorf("getting checkpoint: %w", err)
	}
	return &retCkpt1, nil
}

// InsertModel inserts the model into the database.
func InsertModel(ctx context.Context, name string, description string, metadata []byte,
	labels string, notes string, userID model.UserID, workspaceID int,
) (*modelv1.Model, error) {
	mod := modelv1.Model{}
	q := Bun().NewInsert().
		Model(&mod).
		ExcludeColumn("num_versions", "username", "archived", "id").
		Value("name", "?", name).
		Value("description", "?", description).
		Value("metadata", "?::json", string(metadata)).
		Value("labels", "string_to_array(?, ',')", labels).
		Value("notes", "?", notes).
		Value("user_id", "?", userID).
		Value("workspace_id", "?", workspaceID).
		Value("creation_time", "current_timestamp").
		Value("last_updated_time", "current_timestamp").
		Returning("*")

	err := Bun().NewSelect().
		With("m", q).
		Table("m").
		Column("m.name").
		Column("m.description").
		Column("m.workspace_id").
		Column("m.notes").
		Column("m.metadata").
		ColumnExpr("array_to_json(m.labels) AS labels").
		Column("u.username").
		ColumnExpr("proto_time(m.creation_time) as creation_time").
		ColumnExpr("proto_time(m.last_updated_time) as last_updated_time").
		Column("m.id").
		Join("JOIN users u ON u.id = m.user_id").Scan(ctx, &mod)
	if err != nil {
		return nil, err
	}
	return &mod, err
}

// InsertModelVersion inserts the model version into the database.
func InsertModelVersion(ctx context.Context, id int32, ckptID string, name string, comment string,
	metadata []byte, labels string, notes string, userID model.UserID,
) (*modelv1.ModelVersion, error) {
	modVer := modelv1.ModelVersion{}
	mv := Bun().NewInsert().
		Model(&modVer).
		ExcludeColumn("model", "checkpoint", "username", "id").
		Value("model_id", "?", id).
		Value("version", "(SELECT COALESCE(MAX(version), 0) + 1 FROM model_versions WHERE model_id = ?)", id).
		Value("checkpoint_uuid", "?::uuid", ckptID).
		Value("name", "?", name).
		Value("comment", "?", comment).
		Value("metadata", "?::json", string(metadata)).
		Value("labels", "string_to_array(?, ',')", labels).
		Value("notes", "?", notes).
		Value("user_id", "?", userID).
		Value("creation_time", "current_timestamp").
		Value("last_updated_time", "current_timestamp").
		Returning("*")
	log.Print(mv)

	u := Bun().NewSelect().
		Table("users").
		Column("username").
		Where("id = ?", userID)
	log.Print(u)

	m := Bun().NewSelect().
		TableExpr("models as m").
		Column("m.id").
		Column("m.name").
		Column("m.description").
		Column("m.notes").
		Column("m.metadata").
		ColumnExpr("proto_time(m.creation_time) as creation_time").
		ColumnExpr("proto_time(m.last_updated_time) as last_updated_time").
		ColumnExpr("array_to_json(m.labels) AS labels").
		Column("u.username").
		Column("m.archived").
		ColumnExpr("COUNT(mv.version) AS num_versions").
		Join("JOIN users AS u ON u.id = m.user_id").
		Join(" LEFT JOIN model_versions AS mv ON mv.model_id = m.id").
		Where("m.id = ?", id).
		Group("m.id", "u.id")
	log.Print(m)

	c := Bun().NewSelect().
		TableExpr("proto_checkpoints_view as c").
		ColumnExpr("proto_time(c.report_time) as report_time").
		Column("c.task_id").
		Column("c.allocation_id").
		Column("c.uuid").
		Column("c.resources").
		Column("c.metadata").
		ColumnExpr(bunutils.ProtoStateDBCaseString(checkpointv1.State_value, "c.state", "state",
			"")).
		Column("c.training").
		Column("c.storage_id").
		Where("c.uuid IN (SELECT checkpoint_uuid FROM mv)")
	log.Print(c)

	err := Bun().NewSelect().
		With("mv", mv).
		With("u", u).
		With("m", m).
		With("c", c).
		Table("c", "mv", "m", "u").
		ColumnExpr("TO_JSON(c) AS checkpoint").
		ColumnExpr("TO_JSON(m) AS model").
		ColumnExpr("ARRAY_TO_JSON(mv.labels) AS labels").
		Column("mv.version").
		Column("mv.id").
		ColumnExpr("proto_time(mv.creation_time) as creation_time").
		Column("mv.name").
		Column("mv.comment").
		Column("mv.metadata").
		Column("u.username").
		Where("c.uuid = mv.checkpoint_uuid").Scan(ctx, &modVer)
	if err != nil {
		return nil, err
	}
	return &modVer, err
}
