package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

const (
	asc  = "" // This is blank because ascending is the default
	desc = "DESC"
	max  = "max"
	min  = "min"
)

// ProjectByName returns a project's ID if it exists in the given workspace.
func (db *PgDB) ProjectByName(workspaceName string, projectName string) (int, error) {
	w := workspacev1.Workspace{}
	err := db.Query("get_workspace_from_name", &w, workspaceName)
	if err != nil {
		return 1, err
	}
	p := projectv1.Project{}
	err = db.Query("get_project_from_name", &p, w.Id, projectName)
	if err != nil {
		return 1, err
	}
	if p.Id < 1 {
		return 1, ErrNotFound
	}
	if p.Archived {
		return 1, fmt.Errorf("given workspace or project is archived and cannot add new experiments")
	}
	return int(p.Id), nil
}

// ProjectExperiments returns a list of experiments within a project.
func (db *PgDB) ProjectExperiments(id int) (experiments []*model.Experiment, err error) {
	rows, err := db.sql.Queryx(`
SELECT e.id, state, config, model_definition, start_time, end_time, archived,
	   git_remote, git_commit, git_committer, git_commit_date, owner_id, notes,
		 job_id, u.username as username, project_id, unmanaged
FROM experiments e
JOIN users u ON (e.owner_id = u.id)
WHERE e.project_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var exp model.Experiment
		if err = rows.StructScan(&exp); err != nil {
			return nil, errors.Wrap(err, "unable to read experiment from db")
		}
		experiments = append(experiments, &exp)
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
	return Bun().NewSelect().Table("experiments").
		Where("id IN (?)", bun.In(experimentIDs)).
		Where("state NOT IN (?)", bun.In(model.StatesToStrings(model.TerminalStates))).
		Count(ctx)
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
	JSONKey := model.TrialMetricsJSONPath(metricGroup == model.ValidationMetricGroup)

	err = BunSelectMetricsQuery(metricGroup, false).
		TableExpr("trials t").
		Join("INNER JOIN metrics m ON t.id=m.trial_id").
		ColumnExpr("m.total_batches AS batches_processed, max(t.end_time) as end_time").
		Where("t.experiment_id = ?", experimentID).
		Where(fmt.Sprintf("m.metrics->'%s' ? '%s'", JSONKey, metricName)).
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

// TrainingTrialsSnapshot returns a training metric across each trial in an experiment at a
// specific point of progress.
func (db *PgDB) TrainingTrialsSnapshot(experimentID int, minBatches int, maxBatches int,
	metricName string, startTime time.Time) (trials []*apiv1.TrialsSnapshotResponse_Trial,
	endTime time.Time, err error,
) {
	var rows []snapshotWrapper
	err = db.queryRows(`
SELECT
  t.id AS trial_id,
  t.hparams AS hparams,
  (s.metrics->'avg_metrics'->>$1)::float8 AS metric,
  s.end_time AS end_time,
  s.total_batches as batches
FROM trials t
  INNER JOIN steps s ON t.id=s.trial_id
WHERE t.experiment_id=$2
  AND s.total_batches>=$3
  AND s.total_batches<=$4
  AND s.metrics->'avg_metrics'->$1 IS NOT NULL
  AND s.end_time > $5
ORDER BY s.end_time;`, &rows, metricName, experimentID, minBatches, maxBatches, startTime)
	if err != nil {
		return nil, endTime, errors.Wrapf(err,
			"failed to get snapshot for experiment %d and training metric %s",
			experimentID, metricName)
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

// ValidationTrialsSnapshot returns a training metric across each trial in an experiment at a
// specific point of progress.
func (db *PgDB) ValidationTrialsSnapshot(experimentID int, minBatches int, maxBatches int,
	metricName string, startTime time.Time) (trials []*apiv1.TrialsSnapshotResponse_Trial,
	endTime time.Time, err error,
) {
	var rows []snapshotWrapper
	err = db.queryRows(`
SELECT
  t.id AS trial_id,
  t.hparams AS hparams,
  (v.metrics->'validation_metrics'->>$1)::float8 AS metric,
  v.end_time AS end_time,
  v.total_batches as batches
FROM trials t
JOIN validations v ON t.id = v.trial_id
WHERE t.experiment_id=$2
  AND v.total_batches>=$3
  AND v.total_batches<=$4
  AND v.metrics->'validation_metrics'->$1 IS NOT NULL
  AND v.end_time > $5
ORDER BY v.end_time;`, &rows, metricName, experimentID, minBatches, maxBatches, startTime)
	if err != nil {
		return nil, endTime, errors.Wrapf(err,
			"failed to get snapshot for experiment %d and validation metric %s",
			experimentID, metricName)
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

// CheckExperimentExists checks if the experiment exists.
func (db *PgDB) CheckExperimentExists(id int) (bool, error) {
	var exists bool
	err := db.sql.QueryRow(`
SELECT
EXISTS(
  select id
  FROM experiments
  WHERE id = $1
)`, id).Scan(&exists)
	return exists, err
}

// CheckTrialExists checks if the trial exists.
func (db *PgDB) CheckTrialExists(id int) (bool, error) {
	var exists bool
	err := db.sql.QueryRow(`
SELECT
EXISTS(
  select id
  FROM trials
  WHERE id = $1
)`, id).Scan(&exists)
	return exists, err
}

// TrialExperimentAndRequestID returns the trial's experiment and request ID.
func (db *PgDB) TrialExperimentAndRequestID(id int) (int, model.RequestID, error) {
	var eID int
	var rID model.RequestID
	err := db.sql.QueryRow(`
SELECT e.id, t.request_id
FROM trials t, experiments e
WHERE t.experiment_id = e.id
  AND t.id = $1`, id).Scan(&eID, &rID)
	switch {
	case err == sql.ErrNoRows:
		return eID, rID, errors.WithStack(ErrNotFound)
	case err != nil:
		return eID, rID, errors.Wrap(err, "failed to get trial exp and req id")
	default:
		return eID, rID, nil
	}
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
	experiment *model.Experiment, activeConfig expconf.ExperimentConfig,
) (err error) {
	return AddExperiment(context.TODO(), experiment, activeConfig)
}

// AddExperiment adds the experiment to the database and sets its ID.
func AddExperiment(
	ctx context.Context, experiment *model.Experiment, activeConfig expconf.ExperimentConfig,
) (err error) {
	return Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return AddExperimentTx(ctx, tx, experiment, activeConfig, false)
	})
}

// AddExperimentTx adds the experiment to the database and sets its ID.
func AddExperimentTx(
	ctx context.Context, idb bun.IDB,
	experiment *model.Experiment, activeConfig expconf.ExperimentConfig,
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
	if err = AddProjectHyperparameters(
		ctx, idb, int32(experiment.ProjectID), []int32{int32(experiment.ID)}); err != nil {
		return errors.Wrapf(err, "error updating hyperparameters")
	}

	return nil
}

// RemoveProjectHyperparameters take a list of experiment ids,
// recalculate their respective project hyper parameters.
func RemoveProjectHyperparameters(ctx context.Context, idb bun.IDB, experimentIDs []int32) error {
	if idb == nil {
		idb = Bun()
	}
	var projectIDs []int
	err := idb.NewRaw(`WITH recursive flat (project_id, key, value) AS (
		SELECT project_id, key, value
		FROM experiments,
		jsonb_each(config -> 'hyperparameters')
		WHERE project_id IN (SELECT project_id WHERE id IN (?)) AND id NOT IN (?)
	UNION
		SELECT f.project_id, concat(f.key, '.', j.key), j.value
		FROM flat f,
		jsonb_each(f.value) j
		WHERE jsonb_typeof(f.value) = 'object' AND f.value -> 'type' IS NULL
	), flatten AS (
	SELECT project_id, array_to_json(array_agg(DISTINCT key)) AS data
	FROM flat
	WHERE value -> 'type' IS NOT NULL
	GROUP BY project_id), reset_hp AS (
        UPDATE projects SET hyperparameters = '[]'::jsonb
		WHERE id IN (SELECT project_id FROM experiments WHERE id IN (?))
    )
	UPDATE projects SET hyperparameters = flatten.data FROM flatten
	WHERE flatten.project_id = projects.id`,
		bun.In(experimentIDs), bun.In(experimentIDs), bun.In(experimentIDs)).Scan(ctx, &projectIDs)
	if err != nil {
		return err
	}
	if len(projectIDs) > 1 {
		return errors.New("error removing experiment hyperparameters")
	}
	return nil
}

// AddProjectHyperparameters takes a list of project ids,
// combine their hyper parameters with existing one.
func AddProjectHyperparameters(
	ctx context.Context, idb bun.IDB, projectID int32, experimentIDs []int32,
) error {
	if idb == nil {
		idb = Bun()
	}
	var projectIDs []int
	err := idb.NewRaw(`WITH recursive flat (key, value) AS (
		SELECT key, value
		FROM experiments,
		jsonb_each(config -> 'hyperparameters')
		WHERE id IN (?)
	UNION
		SELECT concat(f.key, '.', j.key), j.value
		FROM flat f,
		jsonb_each(f.value) j
		WHERE jsonb_typeof(f.value) = 'object' AND f.value -> 'type' IS NULL
	), flatten AS (
	SELECT key AS data
	FROM flat WHERE value -> 'type' IS NOT NULL
	UNION SELECT jsonb_array_elements_text(hyperparameters) FROM projects WHERE id = ?
	), agg AS (
		SELECT array_to_json(array_agg(DISTINCT flatten.data)) AS adata FROM flatten
	)
	UPDATE "projects" SET hyperparameters = agg.adata FROM agg WHERE (id = ?) RETURNING id`,
		bun.In(experimentIDs), projectID, projectID).Scan(ctx, &projectIDs)
	if err != nil {
		return err
	}
	if len(projectIDs) > 1 {
		return errors.New("error adding experiment hyperparameters")
	}
	return nil
}

// ExperimentByID looks up an experiment by ID in a database, returning an error if none exists.
func ExperimentByID(ctx context.Context, expID int) (*model.Experiment, error) {
	var experiment model.Experiment

	if err := Bun().NewRaw(`
SELECT e.id, state, config, model_definition, start_time, end_time, archived,
	   git_remote, git_commit, git_committer, git_commit_date, owner_id, notes,
		 job_id, u.username as username, project_id, unmanaged
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
SELECT e.id, e.state, e.config, e.model_definition, e.start_time, e.end_time, e.archived,
       e.git_remote, e.git_commit, e.git_committer, e.git_commit_date, e.owner_id, e.notes,
       e.job_id, u.username as username, e.project_id, unmanaged
FROM experiments e
JOIN trials t ON e.id = t.experiment_id
JOIN users u ON (e.owner_id = u.id)
WHERE t.id = ?`, trialID).Scan(ctx, &experiment); err != nil {
		return nil, MatchSentinelError(err)
	}

	return &experiment, nil
}

// ExperimentByTaskID looks up an experiment by a given taskID, returning an error
// if none exists.
func ExperimentByTaskID(
	ctx context.Context, taskID model.TaskID,
) (*model.Experiment, error) {
	var experiment model.Experiment
	if err := Bun().NewRaw(`
SELECT e.id, e.state, e.config, e.model_definition, e.start_time,
       e.end_time, e.archived, e.git_remote, e.git_commit, e.git_committer, e.git_commit_date,
       e.owner_id, e.notes, e.job_id, u.username as username, e.project_id, e.unmanaged
FROM experiments e
JOIN trials t ON e.id = t.experiment_id
JOIN trial_id_task_id ON t.id = trial_id_task_id.trial_id
JOIN users u ON e.owner_id = u.id
WHERE trial_id_task_id.task_id = ?`, taskID).Scan(ctx, &experiment); err != nil {
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
	SELECT e.id, state, config, model_definition, start_time, end_time, archived,
	git_remote, git_commit, git_committer, git_commit_date, owner_id, notes,
		job_id, u.username as username, project_id, unmanaged
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
SELECT e.id, state, config, model_definition, start_time, end_time, archived,
       git_remote, git_commit, git_committer, git_commit_date, owner_id, job_id,
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
		if model.StoppingStates[exp.State] {
			finalState := model.StoppingToTerminalStates[exp.State]
			if err := db.TerminateExperimentInRestart(exp.ID, finalState); err != nil {
				log.WithError(err).Errorf("finalizing %v on restart", exp)
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
		`UPDATE trials SET state=$1, end_time=$2 WHERE experiment_id=$3 and end_time IS NULL`,
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
		return errors.New("could not transition experiment")
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

	var deletedIDs []int32
	_, err := Bun().NewDelete().Model(&deletedIDs).Table("experiments").
		Where("id IN (?)", bun.In(ids)).
		Returning("id").
		Exec(ctx)
	if err != nil {
		return errors.Wrapf(err, "error deleting experiments %v", ids)
	}
	if len(deletedIDs) != len(ids) {
		return errors.Errorf("mis-match in delete-able experiments versus requested %v", ids)
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

// ExperimentTotalStepTime returns the total elapsed time for all allocations of the experiment
// with the given ID. Any step with a NULL end_time does not contribute. Elapsed time is
// expressed as a floating point number of seconds.
func (db *PgDB) ExperimentTotalStepTime(id int) (float64, error) {
	var seconds float64
	if err := db.sql.Get(&seconds, `
SELECT COALESCE(extract(epoch from sum(a.end_time - a.start_time)), 0)
FROM allocations a
JOIN trial_id_task_id tasks ON a.task_id = tasks.task_id
JOIN trials t ON tasks.trial_id = t.id
WHERE t.experiment_id = $1
`, id); err != nil {
		return 0, errors.Wrapf(err, "querying for total step time of experiment %v", id)
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
func ExperimentsTrialAndTaskIDs(ctx context.Context, idb bun.IDB, expIDs []int) ([]int,
	[]model.TaskID, error,
) {
	if len(expIDs) == 0 {
		return nil, nil, nil
	}

	var res []model.TrialTaskID
	if err := idb.NewSelect().Model(&res).
		Join("JOIN trials ON trials.id = trial_task_id.trial_id").
		Where("trials.experiment_id IN (?)", bun.In(expIDs)).
		Scan(ctx); err != nil {
		return nil, nil, fmt.Errorf("querying for trial / task IDs of experiments %v: %w", expIDs, err)
	}

	var taskIDs []model.TaskID
	trialIDsMap := make(map[int]bool)
	for _, r := range res {
		trialIDsMap[r.TrialID] = true
		taskIDs = append(taskIDs, r.TaskID)
	}

	return maps.Keys(trialIDsMap), taskIDs, nil
}

// ExperimentNumSteps returns the total number of steps for all trials of the experiment.
func (db *PgDB) ExperimentNumSteps(id int) (int64, error) {
	var numSteps int64
	if err := db.sql.Get(&numSteps, `
SELECT count(*)
FROM raw_steps s, trials t
WHERE t.experiment_id = $1 AND s.trial_id = t.id
`, id); err != nil {
		return 0, errors.Wrapf(err, "querying for number of steps of experiment %v", id)
	}
	return numSteps, nil
}

// ExperimentModelDefinitionRaw returns the zipped model definition for an experiment as a byte
// array.
func (db *PgDB) ExperimentModelDefinitionRaw(id int) ([]byte, error) {
	return db.rawQuery(`
SELECT model_definition
FROM experiments
WHERE id = $1`, id)
}

// ExperimentCheckpointsToGCRaw returns a comma-separated string describing checkpoints
// that should be GCed according to the given GC policy parameters. If the delete parameter is true,
// the returned checkpoints are also marked as deleted in the database.
func (db *PgDB) ExperimentCheckpointsToGCRaw(
	id int,
	experimentBest, trialBest, trialLatest int,
) ([]uuid.UUID, error) {
	// The string for the CTEs that we need whether or not we're not deleting the results. The
	// "selected_checkpoints" table contains the checkpoints to return as rows, so that we can easily
	// set the corresponding checkpoints to deleted in a separate CTE if we're deleting.
	query := `
WITH const AS (
    SELECT config->'searcher'->>'metric' AS metric_name,
           (CASE
                WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
                THEN 1
                ELSE -1
            END) AS sign
    FROM experiments WHERE id = $1
), selected_checkpoints AS (
	SELECT c.uuid,
		-- The order includes the id to prevent different rows from having the same
		-- rank, which could cause more than the desired number of checkpoints to be
		-- left out of the result set. Also, any rows with null validation values
		-- will sort to the end, thereby not affecting the ranks of rows with
		-- non-null validations, and will be filtered out later.
		rank() OVER (
			ORDER BY const.sign * (v.metrics->'validation_metrics'->>const.metric_name)::float8
			ASC NULLS LAST, v.id ASC
		) AS experiment_rank,
		rank() OVER (
			PARTITION BY v.trial_id
			ORDER BY const.sign * (v.metrics->'validation_metrics'->>const.metric_name)::float8
			ASC NULLS LAST, v.id ASC
		) AS trial_rank,
		rank() OVER (
			PARTITION BY v.trial_id
			ORDER BY (c.metadata->>'steps_completed')::int DESC
		) AS trial_order_rank,
		v.metrics->'validation_metrics'->>const.metric_name as val_metric
	FROM checkpoints_v2 c
	JOIN const ON true
	JOIN trial_id_task_id ON c.task_id = trial_id_task_id.task_id
    JOIN trials t ON trial_id_task_id.trial_id = t.id
	LEFT JOIN validations v ON v.total_batches = (c.metadata->>'steps_completed')::int AND
		v.trial_id = t.id
	WHERE c.report_time IS NOT NULL
		AND (SELECT COUNT(*) FROM trials t WHERE t.warm_start_checkpoint_id = c.id) = 0
		AND t.experiment_id = $1
)
SELECT sc.uuid AS ID
FROM selected_checkpoints sc
WHERE ((experiment_rank > $2 AND trial_rank > $3) OR (val_metric IS NULL))
	AND trial_order_rank > $4;`

	var checkpointIDRows []struct {
		ID uuid.UUID
	}

	if err := db.queryRows(query, &checkpointIDRows,
		id, experimentBest, trialBest, trialLatest); err != nil {
		return nil, fmt.Errorf(
			"querying for checkpoints that can be deleted according to the GC policy: %w", err)
	}

	var checkpointIDs []uuid.UUID
	for _, cRow := range checkpointIDRows {
		checkpointIDs = append(checkpointIDs, cRow.ID)
	}

	registeredCheckpoints, err := db.GetRegisteredCheckpoints(checkpointIDs)
	if err != nil {
		return nil, err
	}
	var deleteCheckpoints []uuid.UUID
	for _, cUUID := range checkpointIDs {
		if _, ok := registeredCheckpoints[cUUID]; !ok { // not a model registry checkpoint
			deleteCheckpoints = append(deleteCheckpoints, cUUID)
		}
	}

	return deleteCheckpoints, nil
}
