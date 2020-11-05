package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
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

// MetricNames returns the set of training and validation metric names that have been recorded for
// an experiment.
func (db *PgDB) MetricNames(experimentID int, sStartTime time.Time, vStartTime time.Time) (
	training []string, validation []string, sEndTime time.Time, vEndTime time.Time, err error) {
	type namesWrapper struct {
		Name    string    `db:"name"`
		EndTime time.Time `db:"end_time"`
	}

	rows, err := db.sql.Queryx(`
SELECT 
  jsonb_object_keys(s.metrics->'avg_metrics') AS name,
  max(s.end_time) AS end_time
FROM trials t
INNER JOIN steps s ON t.id=s.trial_id
WHERE t.experiment_id=$1
  AND s.end_time > $2
GROUP BY name;`, experimentID, sStartTime)
	if err != nil {
		return nil, nil, sEndTime, vEndTime, errors.Wrapf(err,
			"error querying training metric names for experiment %d", experimentID)
	}
	defer rows.Close()

	for rows.Next() {
		var row namesWrapper
		err = rows.StructScan(&row)
		if err != nil {
			return nil, nil, sEndTime, vEndTime, errors.Wrapf(err,
				"error scanning training metric names for experiment %d",
				experimentID)
		}
		training = append(training, row.Name)
		if row.EndTime.After(sEndTime) {
			sEndTime = row.EndTime
		}
	}

	rows, err = db.sql.Queryx(`
SELECT
  jsonb_object_keys(v.metrics->'validation_metrics') AS name,
  max(v.end_time) AS end_time
FROM trials t
INNER JOIN steps s ON t.id=s.trial_id
LEFT OUTER JOIN validations v ON s.id=v.step_id AND s.trial_id=v.trial_id
WHERE t.experiment_id=$1
  AND v.end_time > $2
GROUP BY name;`, experimentID, vStartTime)
	if err != nil {
		return nil, nil, sEndTime, vEndTime, errors.Wrapf(err,
			"error querying validation metric names for experiment %d", experimentID)
	}
	defer rows.Close()

	for rows.Next() {
		var row namesWrapper
		err = rows.StructScan(&row)
		if err != nil {
			return nil, nil, sEndTime, vEndTime, errors.Wrapf(err,
				"error scanning validation metric names for experiment %d",
				experimentID)
		}
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
	query := `
SELECT (s.prior_batches_processed + num_batches) AS batches_processed,
  max(s.end_time) as end_time
FROM trials t INNER JOIN steps s ON t.id=s.trial_id
WHERE t.experiment_id=$1
  AND s.state = 'COMPLETED'
  AND s.metrics->'avg_metrics' ? $2
  AND s.end_time > $3
GROUP BY batches_processed;`

	var rows []*batchesWrapper
	err = db.queryRows(query, &rows, experimentID, metricName, startTime)
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
	query := `
SELECT (s.prior_batches_processed + num_batches) AS batches_processed,
  max(v.end_time) as end_time
FROM trials t INNER JOIN steps s ON t.id=s.trial_id
  LEFT OUTER JOIN validations v ON s.id=v.step_id AND s.trial_id=v.trial_id
WHERE t.experiment_id=$1
  AND v.state = 'COMPLETED'
  AND v.metrics->'validation_metrics' ? $2
  AND v.end_time > $3
GROUP BY batches_processed;`

	var rows []*batchesWrapper
	err = db.queryRows(query, &rows, experimentID, metricName, startTime)
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
