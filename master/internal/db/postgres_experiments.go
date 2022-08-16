package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/lttb"
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
		 job_id, u.username as username
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

// MetricNames returns the set of training and validation metric names that have been recorded for
// an experiment.
func (db *PgDB) MetricNames(experimentID int, sStartTime time.Time, vStartTime time.Time) (
	training []string, validation []string, sEndTime time.Time, vEndTime time.Time, err error,
) {
	type namesWrapper struct {
		Name    string    `db:"name"`
		EndTime time.Time `db:"end_time"`
	}

	var rows []namesWrapper
	err = db.queryRows(`
SELECT
  jsonb_object_keys(s.metrics->'avg_metrics') AS name,
  max(s.end_time) AS end_time
FROM trials t
JOIN steps s ON t.id=s.trial_id
WHERE t.experiment_id=$1
  AND s.end_time > $2
GROUP BY name`, &rows, experimentID, sStartTime)
	if err != nil {
		return nil, nil, sEndTime, vEndTime, errors.Wrapf(err,
			"error querying training metric names for experiment %d", experimentID)
	}
	for _, row := range rows {
		training = append(training, row.Name)
		if row.EndTime.After(sEndTime) {
			sEndTime = row.EndTime
		}
	}

	err = db.queryRows(`
SELECT
  jsonb_object_keys(v.metrics->'validation_metrics') AS name,
  max(v.end_time) AS end_time
FROM trials t
JOIN validations v ON t.id = v.trial_id
WHERE t.experiment_id=$1
  AND v.end_time > $2
GROUP BY name`, &rows, experimentID, vStartTime)
	if err != nil {
		return nil, nil, sEndTime, vEndTime, errors.Wrapf(err,
			"error querying validation metric names for experiment %d", experimentID)
	}
	for _, row := range rows {
		validation = append(validation, row.Name)
		if row.EndTime.After(sEndTime) {
			sEndTime = row.EndTime
		}
	}

	return training, validation, sEndTime, vEndTime, err
}

// ExpCompareMetricNames returns the set of training and validation metric names
// that have been recorded for a list of trials.
func (db *PgDB) ExpCompareMetricNames(trialIDs []int32, sStartTime time.Time,
	vStartTime time.Time) (training []string, validation []string, sEndTime time.Time,
	vEndTime time.Time, err error,
) {
	type namesWrapper struct {
		Name    string    `db:"name"`
		EndTime time.Time `db:"end_time"`
	}
	var rows []namesWrapper
	err = db.queryRows(`
SELECT
  jsonb_object_keys(s.metrics->'avg_metrics') AS name,
  max(s.end_time) AS end_time
FROM trials t
JOIN steps s ON t.id=s.trial_id
WHERE t.id IN (SELECT unnest($1::int [])::int)
  AND s.end_time > $2
GROUP BY name`, &rows, trialIDs, sStartTime)
	if err != nil {
		return nil, nil, sEndTime, vEndTime, errors.Wrapf(err,
			"error querying training metric names fort rials")
	}
	for _, row := range rows {
		training = append(training, row.Name)
		if row.EndTime.After(sEndTime) {
			sEndTime = row.EndTime
		}
	}

	err = db.queryRows(`
SELECT
  jsonb_object_keys(v.metrics->'validation_metrics') AS name,
  max(v.end_time) AS end_time
FROM trials t
JOIN validations v ON t.id = v.trial_id
WHERE t.id IN (SELECT unnest($1::int [])::int)
  AND v.end_time > $2
GROUP BY name`, &rows, trialIDs, vStartTime)
	if err != nil {
		return nil, nil, sEndTime, vEndTime, errors.Wrapf(err,
			"error querying validation metric names for trials")
	}
	for _, row := range rows {
		validation = append(validation, row.Name)
		if row.EndTime.After(sEndTime) {
			sEndTime = row.EndTime
		}
	}
	return training, validation, sEndTime, vEndTime, err
}

type batchesWrapper struct {
	Batches int32     `db:"batches_processed"`
	EndTime time.Time `db:"end_time"`
}

// TrainingMetricBatches returns the milestones (in batches processed) at which a specific training
// metric was recorded.
func (db *PgDB) TrainingMetricBatches(experimentID int, metricName string, startTime time.Time) (
	batches []int32, endTime time.Time, err error,
) {
	var rows []*batchesWrapper
	err = db.queryRows(`
SELECT s.total_batches AS batches_processed,
  max(s.end_time) as end_time
FROM trials t INNER JOIN steps s ON t.id=s.trial_id
WHERE t.experiment_id=$1
  AND s.state = 'COMPLETED'
  AND s.metrics->'avg_metrics' ? $2
  AND s.end_time > $3
GROUP BY batches_processed;`, &rows, experimentID, metricName, startTime)
	if err != nil {
		return nil, endTime, errors.Wrapf(err, "error querying DB for training metric batches")
	}
	for _, row := range rows {
		batches = append(batches, row.Batches)
		if row.EndTime.After(endTime) {
			endTime = row.EndTime
		}
	}

	return batches, endTime, nil
}

// ValidationMetricBatches returns the milestones (in batches processed) at which a specific
// validation metric was recorded.
func (db *PgDB) ValidationMetricBatches(experimentID int, metricName string, startTime time.Time) (
	batches []int32, endTime time.Time, err error,
) {
	var rows []*batchesWrapper
	err = db.queryRows(`
SELECT
  v.total_batches AS batches_processed,
  max(v.end_time) as end_time
FROM trials t
JOIN validations v ON t.id = v.trial_id
WHERE t.experiment_id=$1
  AND v.state = 'COMPLETED'
  AND v.metrics->'validation_metrics' ? $2
  AND v.end_time > $3
GROUP BY batches_processed`, &rows, experimentID, metricName, startTime)
	if err != nil {
		return nil, endTime, errors.Wrapf(err, "error querying DB for validation metric batches")
	}
	for _, row := range rows {
		batches = append(batches, row.Batches)
		if row.EndTime.After(endTime) {
			endTime = row.EndTime
		}
	}

	return batches, endTime, nil
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
func (db *PgDB) TopTrialsByMetric(experimentID int, maxTrials int, metric string,
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
    %s((v.metrics->'validation_metrics'->>$1)::float8) as best_metric
  FROM trials t
  JOIN validations v ON t.id = v.trial_id
  WHERE t.experiment_id=$2
    AND v.state = 'COMPLETED'
  GROUP BY t.id
  ORDER BY best_metric %s
  LIMIT $3
) t;`, aggregate, order), metric, experimentID, maxTrials)
	return trials, err
}

// ExpCompareTopTrialsByMetric chooses the subset of trials from a list of experiments
// that recorded the best values for the specified metric at any point during the trial.
func (db *PgDB) ExpCompareTopTrialsByMetric(experimentIDs []int32, maxTrials int, metric string,
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
    %s((v.metrics->'validation_metrics'->>$1)::float8) as best_metric
  FROM trials t
  JOIN validations v ON t.id = v.trial_id
  WHERE t.experiment_id in (SELECT unnest($2::int [])::int)
    AND v.state = 'COMPLETED'
  GROUP BY t.id
  ORDER BY best_metric %s
  LIMIT $3
) t;`, aggregate, order), metric, experimentIDs, maxTrials)
	return trials, err
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
    AND v.state = 'COMPLETED'
  GROUP BY t.id
  ORDER BY progress DESC, best_metric %s
  LIMIT $3
) t;`, aggregate, order), metric, experimentID, maxTrials)
	return trials, err
}

func scanMetricsSeries(metricSeries []lttb.Point, rows *sql.Rows) ([]lttb.Point, time.Time) {
	var maxEndTime time.Time
	for rows.Next() {
		var batches uint
		var value float64
		var endTime time.Time
		err := rows.Scan(&batches, &value, &endTime)
		if err != nil {
			// Could be a bad metric name, sparse metric, nested type, etc.
			continue
		}
		metricSeries = append(metricSeries, lttb.Point{X: float64(batches), Y: value})
		if endTime.After(maxEndTime) {
			maxEndTime = endTime
		}
	}
	return metricSeries, maxEndTime
}

// TrainingMetricsSeries returns a time-series of the specified training metric in the specified
// trial.
func (db *PgDB) TrainingMetricsSeries(trialID int32, startTime time.Time, metricName string,
	startBatches int, endBatches int) (metricSeries []lttb.Point, maxEndTime time.Time,
	err error,
) {
	rows, err := db.sql.Query(`
SELECT
  total_batches AS batches,
  s.metrics->'avg_metrics'->$1 AS value,
  s.end_time as end_time
FROM trials t
  INNER JOIN steps s ON t.id=s.trial_id
WHERE t.id=$2
  AND s.state = 'COMPLETED'
  AND total_batches >= $3
  AND total_batches <= $4
  AND s.end_time > $5
  AND s.metrics->'avg_metrics'->$1 IS NOT NULL
ORDER BY batches;`, metricName, trialID, startBatches, endBatches, startTime)
	if err != nil {
		return nil, maxEndTime, errors.Wrapf(err, "failed to get metrics to sample for experiment")
	}
	defer rows.Close()
	metricSeries, maxEndTime = scanMetricsSeries(metricSeries, rows)
	return metricSeries, maxEndTime, nil
}

// ValidationMetricsSeries returns a time-series of the specified validation metric in the specified
// trial.
func (db *PgDB) ValidationMetricsSeries(trialID int32, startTime time.Time, metricName string,
	startBatches int, endBatches int) (metricSeries []lttb.Point, maxEndTime time.Time,
	err error,
) {
	rows, err := db.sql.Query(`
SELECT
  v.total_batches AS batches,
  (v.metrics->'validation_metrics'->>$1)::float8 AS value,
  v.end_time as end_time
FROM trials t
JOIN validations v ON t.id = v.trial_id
WHERE t.id=$2
  AND v.state = 'COMPLETED'
  AND v.total_batches >= $3
  AND v.total_batches <= $4
  AND v.end_time > $5
  AND v.metrics->'validation_metrics'->$1 IS NOT NULL
ORDER BY batches;`, metricName, trialID, startBatches, endBatches, startTime)
	if err != nil {
		return nil, maxEndTime, errors.Wrapf(err, "failed to get metrics to sample for experiment")
	}
	defer rows.Close()
	metricSeries, maxEndTime = scanMetricsSeries(metricSeries, rows)
	return metricSeries, maxEndTime, nil
}

type hpImportanceDataWrapper struct {
	TrialID int     `db:"trial_id"`
	Hparams []byte  `db:"hparams"`
	Batches int     `db:"batches"`
	Metric  float64 `db:"metric"`
}

func unmarshalHPImportanceHParams(r hpImportanceDataWrapper) (model.HPImportanceTrialData, int,
	error,
) {
	entry := model.HPImportanceTrialData{
		TrialID: r.TrialID,
		Metric:  r.Metric,
	}
	return entry, r.Batches, json.Unmarshal(r.Hparams, &entry.Hparams)
}

// FetchHPImportanceTrainingData retrieves all the data needed by the hyperparameter importance
// algorithm to measure the relative importance of various hyperparameters for one specific training
// metric across all the trials in an experiment.
func (db *PgDB) FetchHPImportanceTrainingData(experimentID int, metric string) (
	map[int][]model.HPImportanceTrialData, error,
) {
	var rows []hpImportanceDataWrapper
	results := make(map[int][]model.HPImportanceTrialData)
	// TODO: aren't we ignoring overtraining by taking the last?
	err := db.queryRows(`
SELECT
  t.id AS trial_id,
  t.hparams AS hparams,
  s.total_batches AS batches,
  s.metrics->'avg_metrics'->$1 AS metric
FROM trials t
  INNER JOIN steps s ON t.id=s.trial_id
  INNER JOIN (
    SELECT
      t.id as trial_id,
	  s.total_batches AS total_batches,
	  max(s.total_batches) AS batches
    FROM trials t
  	INNER JOIN steps s ON t.id=s.trial_id
    WHERE t.experiment_id=$2
	  AND s.state = 'COMPLETED'
    GROUP BY t.id, s.total_batches
  ) filter
	ON s.total_batches = filter.total_batches
	AND t.id = filter.trial_id`, &rows, metric, experimentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get training metrics for hyperparameter importance")
	}
	for _, row := range rows {
		result, batches, err := unmarshalHPImportanceHParams(row)
		if err != nil {
			return nil, errors.Wrap(err,
				"failed to process training metrics for hyperparameter importance")
		}
		results[batches] = append(results[batches], result)
	}
	return results, nil
}

// FetchHPImportanceValidationData retrieves all the data needed by the hyperparameter importance
// algorithm to measure the relative importance of various hyperparameters for one specific
// validation metric across all the trials in an experiment.
func (db *PgDB) FetchHPImportanceValidationData(experimentID int, metric string) (
	map[int][]model.HPImportanceTrialData, error,
) {
	var rows []hpImportanceDataWrapper
	results := make(map[int][]model.HPImportanceTrialData)
	err := db.queryRows(`
SELECT
  t.id AS trial_id,
  t.hparams AS hparams,
  v.total_batches AS batches,
  (v.metrics->'validation_metrics'->>$1)::float8 as metric
FROM trials t
JOIN validations v ON t.id = v.trial_id
JOIN (
  SELECT
    t.id as trial_id,
    max(v.total_batches) AS total_batches
  FROM trials t
  JOIN validations v ON t.id = v.trial_id
  WHERE t.experiment_id=$2
    AND v.state = 'COMPLETED'
  GROUP BY t.id, v.total_batches
) filter
ON v.total_batches = filter.total_batches
AND t.id = filter.trial_id`, &rows, metric, experimentID)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to get validation metrics for hyperparameter importance")
	}
	for _, row := range rows {
		result, batches, err := unmarshalHPImportanceHParams(row)
		if err != nil {
			return nil, errors.Wrap(err,
				"Failed to process validation metrics for hyperparameter importance")
		}
		results[batches] = append(results[batches], result)
	}
	return results, nil
}

// GetHPImportance returns the hyperparameter importance data and status for an experiment.
func (db *PgDB) GetHPImportance(experimentID int) (result model.ExperimentHPImportance, err error) {
	var jsonString []byte
	err = db.sql.Get(&jsonString, "SELECT hpimportance FROM experiments WHERE id=$1", experimentID)
	if err != nil {
		return result, errors.Wrap(err, "Error retrieving hyperparameter importance")
	}
	if len(jsonString) > 0 {
		err = json.Unmarshal(jsonString, &result)
		if err != nil {
			return result, errors.Wrap(err, "Error unmarshaling hyperparameter importance")
		}
	}
	if result.TrainingMetrics == nil {
		result.TrainingMetrics = make(map[string]model.MetricHPImportance)
	}
	if result.ValidationMetrics == nil {
		result.ValidationMetrics = make(map[string]model.MetricHPImportance)
	}
	return result, err
}

// SetHPImportance writes the current hyperparameter importance data and status to the database.
// It should only be called from the HPImportance manager actor, to ensure coherence. It will set
// hpi.Partial according to the individual metric statuses to facilitate faster querying for any
// incomplete work.
func (db *PgDB) SetHPImportance(experimentID int, value model.ExperimentHPImportance) error {
	value.Partial = false
	for _, metricHpi := range value.TrainingMetrics {
		if metricHpi.Pending || metricHpi.InProgress {
			value.Partial = true
			break
		}
	}
	if !value.Partial {
		for _, metricHpi := range value.ValidationMetrics {
			if metricHpi.Pending || metricHpi.InProgress {
				value.Partial = true
				break
			}
		}
	}
	jsonString, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = db.sql.Exec("UPDATE experiments SET hpimportance=$1 WHERE id=$2",
		jsonString, experimentID)
	return err
}

// GetPartialHPImportance returns all the experiment IDs and their HP importance data if they had
// any pending or in-progress tasks the last time they were written to the DB.
func (db *PgDB) GetPartialHPImportance() ([]int, []model.ExperimentHPImportance, error) {
	type partialHPImportanceRow struct {
		ID           int    `db:"id"`
		HPImportance []byte `db:"hpimportance"`
	}

	var rows []partialHPImportanceRow
	var ids []int
	var hpis []model.ExperimentHPImportance
	err := db.queryRows(`
SELECT id, hpimportance FROM experiments
WHERE (hpimportance->>'partial')::boolean=true`, &rows)
	if err != nil {
		return nil, nil, errors.Wrapf(err,
			"failed to request partial hyperparameter importance work")
	}
	for _, row := range rows {
		var hpi model.ExperimentHPImportance
		err = json.Unmarshal(row.HPImportance, &hpi)
		if err != nil {
			return nil, nil, errors.Wrapf(err,
				"Failed to parse partial hyperparameter importance for experiment %d", row.ID)
		}
		if hpi.TrainingMetrics == nil {
			hpi.TrainingMetrics = make(map[string]model.MetricHPImportance)
		}
		if hpi.ValidationMetrics == nil {
			hpi.ValidationMetrics = make(map[string]model.MetricHPImportance)
		}
		hpis = append(hpis, hpi)
		ids = append(ids, row.ID)
	}
	return ids, hpis, nil
}

// ExperimentBestSearcherValidation returns the best searcher validation for an experiment.
func (db *PgDB) ExperimentBestSearcherValidation(id int) (float32, error) {
	conf, err := db.ExperimentConfig(id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get experiment config")
	}

	metricName := conf.Searcher().Metric()
	metricOrdering := desc
	if conf.Searcher().SmallerIsBetter() {
		metricOrdering = asc
	}

	var metric float32
	switch err := db.sql.QueryRowx(fmt.Sprintf(`
SELECT (v.metrics->'validation_metrics'->>$2)::float8
FROM validations v, trials t
WHERE v.trial_id = t.id
  AND t.experiment_id = $1
  AND v.state = 'COMPLETED'
ORDER BY (v.metrics->'validation_metrics'->>$2)::float8 %s
LIMIT 1`, metricOrdering), id, metricName).Scan(&metric); {
	case errors.Is(err, sql.ErrNoRows):
		return 0, ErrNotFound
	case err != nil:
		return 0, errors.Wrap(err, "querying best experiment validation")
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
func (db *PgDB) AddExperiment(experiment *model.Experiment) (err error) {
	if experiment.ID != 0 {
		return errors.Errorf("error adding an experiment with non-zero id %v", experiment.ID)
	}
	return db.withTransaction("add_experiment", func(tx *sqlx.Tx) error {
		job := model.Job{
			JobID:   experiment.JobID,
			JobType: model.JobTypeExperiment,
			OwnerID: experiment.OwnerID,
		}
		if err := addJob(tx, &job); err != nil {
			return errors.Wrapf(err, "error inserting job %v", job)
		}
		err := namedGet(tx, &experiment.ID, `
	INSERT INTO experiments
	(state, config, model_definition, start_time, end_time, archived, parent_id, progress,
	 git_remote, git_commit, git_committer, git_commit_date, owner_id, original_config, notes, job_id,
 	project_id)
	VALUES (:state, :config, :model_definition, :start_time, :end_time, :archived, :parent_id, 0,
					:git_remote, :git_commit, :git_committer, :git_commit_date, :owner_id, :original_config,
					:notes, :job_id, :project_id)
	RETURNING id`, experiment)
		if err != nil {
			return errors.Wrapf(err, "error inserting experiment %v", *experiment)
		}
		return nil
	})
}

// ExperimentByID looks up an experiment by ID in a database, returning an error if none exists.
func (db *PgDB) ExperimentByID(id int) (*model.Experiment, error) {
	var experiment model.Experiment

	if err := db.query(`
SELECT e.id, state, config, model_definition, start_time, end_time, archived,
	   git_remote, git_commit, git_committer, git_commit_date, owner_id, notes,
		 job_id, u.username as username
FROM experiments e
JOIN users u ON (e.owner_id = u.id)
WHERE e.id = $1`, &experiment, id); err != nil {
		return nil, err
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

// ExperimentWithoutConfigByID looks up an experiment by ID in a database, returning an error if
// none exists. It loads the experiment without its configuration, for callers that do not need
// it, or can't handle backwards incompatible changes.
func (db *PgDB) ExperimentWithoutConfigByID(id int) (*model.Experiment, error) {
	var experiment model.Experiment

	if err := db.query(`
SELECT e.id, state, model_definition, start_time, end_time, archived,
       git_remote, git_commit, git_committer, git_commit_date, owner_id, notes,
			 job_id, u.username as username
FROM experiments e
JOIN users u ON e.owner_id = u.id
WHERE e.id = $1`, &experiment, id); err != nil {
		return nil, err
	}

	return &experiment, nil
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
			 u.username as username
FROM experiments e
JOIN users u ON e.owner_id = u.id
WHERE state IN ('ACTIVE', 'PAUSED', 'STOPPING_CANCELED', 'STOPPING_COMPLETED', 'STOPPING_ERROR')`)
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
			items, err := rows.SliceScan()
			if err != nil {
				return nil, errors.Wrap(err, "unable to read experiment from db")
			}

			expID, ok := items[0].(int64)
			if !ok {
				return nil, errors.Errorf(
					"Expected an integer experiment ID, but got: %s", reflect.TypeOf(items[0]))
			}

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
func (db *PgDB) SaveExperimentConfig(experiment *model.Experiment) error {
	query := `
UPDATE experiments
SET config=:config
WHERE id = :id`
	return db.namedExecOne(query, experiment)
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

// DeleteExperiment deletes an existing experiment.
func (db *PgDB) DeleteExperiment(id int) error {
	return db.withTransaction("delete experiment", func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(`
DELETE FROM raw_steps
WHERE trial_id IN (SELECT id FROM trials WHERE experiment_id = $1)
`, id); err != nil {
			return errors.Wrapf(err, "error deleting steps for experiment %v", id)
		}

		if _, err := tx.Exec(`
DELETE FROM raw_validations
WHERE trial_id IN (SELECT id FROM trials WHERE experiment_id = $1)
`, id); err != nil {
			return errors.Wrapf(err, "error deleting validations for experiment %v", id)
		}

		if _, err := tx.Exec(`
DELETE FROM raw_checkpoints
WHERE trial_id IN (SELECT id FROM trials WHERE experiment_id = $1)
`, id); err != nil {
			return errors.Wrapf(err, "error deleting checkpoints for experiment %v", id)
		}

		if _, err := tx.Exec(`
DELETE FROM checkpoints_v2
WHERE task_id IN (
	SELECT tk.task_id
	FROM tasks tk
	JOIN trials t ON t.task_id = tk.task_id
	JOIN experiments e ON t.experiment_id = e.id
	WHERE experiment_id = $1
)`, id); err != nil {
			return errors.Wrapf(err, "error deleting checkpoints (v2) for experiment %v", id)
		}

		if err := db.deleteSnapshotsForExperiment(id)(tx); err != nil {
			return errors.Wrapf(err, "error deleting snapshots for experiment %v", id)
		}

		if _, err := tx.Exec(`
DELETE FROM trials
WHERE experiment_id = $1;
`, id); err != nil {
			return errors.Wrapf(err, "error deleting trials for experiment %v", id)
		}

		result, err := tx.Exec(`
DELETE FROM experiments
WHERE id = $1
`, id)
		if err != nil {
			return errors.Wrapf(err, "error deleting experiment %v", id)
		}
		switch num, err := result.RowsAffected(); {
		case err != nil:
			return errors.Wrapf(err, "error in RowsAffected when deleting experiment %v", id)
		case num != 1:
			return errors.Errorf("error deleting non-existing experiment %v", id)
		}
		return nil
	})
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

// ExperimentConfig returns the full config object for an experiment.
func (db *PgDB) ExperimentConfig(id int) (expconf.ExperimentConfig, error) {
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
	return schemas.WithDefaults(expConfig).(expconf.ExperimentConfig), nil
}

// ExperimentTotalStepTime returns the total elapsed time for all allocations of the experiment
// with the given ID. Any step with a NULL end_time does not contribute. Elapsed time is
// expressed as a floating point number of seconds.
func (db *PgDB) ExperimentTotalStepTime(id int) (float64, error) {
	var seconds float64
	if err := db.sql.Get(&seconds, `
SELECT COALESCE(extract(epoch from sum(a.end_time - a.start_time)), 0)
FROM allocations a, trials t
WHERE t.experiment_id = $1 AND a.task_id = t.task_id
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

// ExperimentTrialAndTaskIDs returns the trial and task IDs for the experiment.
func (db *PgDB) ExperimentTrialAndTaskIDs(expID int) ([]int, []model.TaskID, error) {
	var trialIDRows []struct {
		ID     int          `db:"id"`
		TaskID model.TaskID `db:"task_id"`
	}
	if err := db.queryRows(`
SELECT id, task_id
FROM trials
WHERE trials.experiment_id = $1
`, &trialIDRows, expID); err != nil {
		return nil, nil, errors.Wrapf(err, "querying for trial IDs of experiment %v", expID)
	}
	var trialIDs []int
	var taskIDs []model.TaskID
	for _, r := range trialIDRows {
		trialIDs = append(trialIDs, r.ID)
		taskIDs = append(taskIDs, r.TaskID)
	}
	return trialIDs, taskIDs, nil
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
    SELECT *
    FROM (
        SELECT *,
               -- The order includes the id to prevent different rows from having the same
               -- rank, which could cause more than the desired number of checkpoints to be
               -- left out of the result set. Also, any rows with null validation values
               -- will sort to the end, thereby not affecting the ranks of rows with
               -- non-null validations, and will be filtered out later.
               rank() OVER (
                   ORDER BY
                       const.sign * (step->'validation'->'metrics'->'validation_metrics'
                                     ->>const.metric_name)::float8 ASC NULLS LAST, id ASC
               ) AS experiment_rank,
               rank() OVER (
                   PARTITION BY trial_id
                   ORDER BY
                       const.sign * (step->'validation'->'metrics'->'validation_metrics'
                                     ->>const.metric_name)::float8 ASC NULLS LAST, id ASC
               ) AS trial_rank,
               rank() OVER (
                   PARTITION BY trial_id
                   ORDER BY total_batches DESC
               ) AS trial_order_rank
        FROM (
            SELECT c.id, c.trial_id, c.steps_completed as total_batches, c.state,
                   c.report_time as end_time, c.uuid, c.resources, c.metadata,
                   (SELECT row_to_json(s)
                    FROM (
                        SELECT s.end_time, s.id, s.state, s.trial_id,
                            s.total_batches,
                            (SELECT row_to_json(v)
                            FROM (
                                SELECT v.end_time, v.id, v.metrics,
                                    v.state, v.total_batches, v.trial_id
                                    FROM validations v
                                    WHERE v.trial_id = t.id AND v.total_batches = s.total_batches
                                ) v
                               ) AS validation
                        FROM steps s
                        WHERE s.total_batches = c.steps_completed AND s.trial_id = c.trial_id
                    ) s
                   ) AS step,
                   -- We later filter out any checkpoints with any corresponding warm start
                   -- trials, so we can just put an empty list here. (TODO(dzhu): This is
                   -- here for backwards compatibility with Python, but could maybe be
                   -- removed.)
                   '[]'::jsonb AS warm_start_trials
            FROM checkpoints_view c, trials t, const
            WHERE c.state = 'COMPLETED' AND c.trial_id = t.id AND t.experiment_id = $1
        ) _, const
    ) c, const
    WHERE (SELECT COUNT(*) FROM trials t WHERE t.warm_start_checkpoint_id = c.id) = 0
          AND c.trial_order_rank > $4
          AND ((c.experiment_rank > $2
                AND c.trial_rank > $3)
               OR (c.step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                   IS NULL))
)
SELECT selected_checkpoints.uuid AS ID from selected_checkpoints;`

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
