package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/lttb"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	asc  = "" // This is blank because ascending is the default
	desc = "DESC"
	max  = "max"
	min  = "min"
)

// ExperimentLabelUsage returns a flattened and deduplicated list of all the
// labels in use across all experiments.
func (db *PgDB) ExperimentLabelUsage() (labelUsage map[string]int, err error) {
	// First, assemble all the JSON lists that the database returns into a
	// single tally of all the labels
	type dbLabelList struct {
		Labels []byte
	}
	var rawLists []dbLabelList
	err = db.Query("get_experiment_labels", &rawLists)
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
	err error) {
	row := db.sql.QueryRow(
		"SELECT state, COALESCE(progress, 0) as progress FROM experiments WHERE id=$1",
		experimentID)
	err = row.Scan(&state, &progress)
	return state, progress, err
}

// MetricNames returns the set of training and validation metric names that have been recorded for
// an experiment.
func (db *PgDB) MetricNames(experimentID int, sStartTime time.Time, vStartTime time.Time) (
	training []string, validation []string, sEndTime time.Time, vEndTime time.Time, err error) {
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
INNER JOIN steps s ON t.id=s.trial_id
WHERE t.experiment_id=$1
  AND s.end_time > $2
GROUP BY name;`, &rows, experimentID, sStartTime)
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
INNER JOIN steps s ON t.id=s.trial_id
LEFT OUTER JOIN validations v ON s.total_batches=v.total_batches AND s.trial_id=v.trial_id
WHERE t.experiment_id=$1
  AND v.end_time > $2
GROUP BY name;`, &rows, experimentID, vStartTime)
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

type batchesWrapper struct {
	Batches int32     `db:"batches_processed"`
	EndTime time.Time `db:"end_time"`
}

// TrainingMetricBatches returns the milestones (in batches processed) at which a specific training
// metric was recorded.
func (db *PgDB) TrainingMetricBatches(experimentID int, metricName string, startTime time.Time) (
	batches []int32, endTime time.Time, err error) {
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
	batches []int32, endTime time.Time, err error) {
	var rows []*batchesWrapper
	err = db.queryRows(`
SELECT s.total_batches AS batches_processed,
  max(v.end_time) as end_time
FROM trials t INNER JOIN steps s ON t.id=s.trial_id
  LEFT OUTER JOIN validations v ON s.total_batches=v.total_batches AND s.trial_id=v.trial_id
WHERE t.experiment_id=$1
  AND v.state = 'COMPLETED'
  AND v.metrics->'validation_metrics' ? $2
  AND v.end_time > $3
GROUP BY batches_processed;`, &rows, experimentID, metricName, startTime)
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
	endTime time.Time, err error) {
	var rows []snapshotWrapper
	err = db.queryRows(`
SELECT
  t.id AS trial_id,
  t.hparams AS hparams,
  s.metrics->'avg_metrics'->$1 AS metric,
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
	endTime time.Time, err error) {
	var rows []snapshotWrapper
	err = db.queryRows(`
SELECT
  t.id AS trial_id,
  t.hparams AS hparams,
  v.metrics->'validation_metrics'->$1 AS metric,
  v.end_time AS end_time,
  s.total_batches as batches
FROM trials t
  INNER JOIN steps s ON t.id=s.trial_id
  LEFT OUTER JOIN validations v ON s.total_batches=v.total_batches AND s.trial_id=v.trial_id
WHERE t.experiment_id=$2
  AND s.total_batches>=$3
  AND s.total_batches<=$4
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
	smallerIsBetter bool) (trials []int32, err error) {
	order := desc
	aggregate := max
	if smallerIsBetter {
		order = asc
		aggregate = min
	}
	err = db.sql.Select(&trials, fmt.Sprintf(`
SELECT t.id FROM (
  SELECT t.id,
    %s((v.metrics->'validation_metrics'->$1)::text::numeric) as best_metric
  FROM trials t
    INNER JOIN steps s ON t.id=s.trial_id
    RIGHT JOIN validations v ON s.total_batches=v.total_batches AND s.trial_id=v.trial_id
  WHERE t.experiment_id=$2
    AND v.state = 'COMPLETED'
  GROUP BY t.id
  ORDER BY best_metric %s
  LIMIT $3
) t;`, aggregate, order), metric, experimentID, maxTrials)
	return trials, err
}

// TopTrialsByTrainingLength chooses the subset of trials that has been training for the highest
// number of batches, using the specified metric as a tie breaker.
func (db *PgDB) TopTrialsByTrainingLength(experimentID int, maxTrials int, metric string,
	smallerIsBetter bool) (trials []int32, err error) {
	order := desc
	aggregate := max
	if smallerIsBetter {
		order = asc
		aggregate = min
	}

	err = db.sql.Select(&trials, fmt.Sprintf(`
SELECT t.id FROM (
  SELECT t.id,
    max(s.total_batches) as progress,
    %s((v.metrics->'validation_metrics'->$1)::text::numeric) as best_metric
  FROM trials t
    INNER JOIN steps s ON t.id=s.trial_id
    RIGHT JOIN validations v ON s.total_batches=v.total_batches AND s.trial_id=v.trial_id
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
	err error) {
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
	err error) {
	rows, err := db.sql.Query(`
SELECT 
  v.total_batches AS batches,
  v.metrics->'validation_metrics'->$1 AS value,
  v.end_time as end_time
FROM trials t
  INNER JOIN steps s ON t.id=s.trial_id
  LEFT OUTER JOIN validations v ON s.total_batches=v.total_batches AND s.trial_id=v.trial_id
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
	error) {
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
	map[int][]model.HPImportanceTrialData, error) {
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
	map[int][]model.HPImportanceTrialData, error) {
	var rows []hpImportanceDataWrapper
	results := make(map[int][]model.HPImportanceTrialData)
	err := db.queryRows(`
SELECT
  t.id AS trial_id,
  t.hparams AS hparams,
  s.total_batches AS batches,
  v.metrics->'validation_metrics'->$1 as metric
FROM trials t
  INNER JOIN steps s ON t.id=s.trial_id
  RIGHT JOIN validations v ON s.total_batches=v.total_batches AND s.trial_id=v.trial_id
  INNER JOIN (
    SELECT
      t.id as trial_id,
      s.total_batches AS total_batches,
      max(s.total_batches) AS batches
    FROM trials t
      INNER JOIN steps s ON t.id=s.trial_id
      RIGHT JOIN validations v ON s.total_batches=v.total_batches AND s.trial_id=v.trial_id
    WHERE t.experiment_id=$2
      AND v.state = 'COMPLETED'
    GROUP BY t.id, s.total_batches
  ) filter
	ON s.total_batches = filter.total_batches
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
