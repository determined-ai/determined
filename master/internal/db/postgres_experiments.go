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
func (db *PgDB) MetricNames(experimentID int, sStartTime time.Time, vStartTime time.Time) (training []string, validation []string, sEndTime time.Time, vEndTime time.Time, err error) {

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

// MetricBatches returns the milestones (in batches processed) at which a specific metric was
// recorded.
func (db *PgDB) MetricBatches(experimentID int, trainingMetric string, validationMetric string,
	startTime time.Time) (batches []int32, endTime time.Time, err error) {
	endTime = startTime
	var metricName string
	const TRAINING = "training"
	const VALIDATION = "validation"
	var metricType string
	if len(trainingMetric) > 0 && len(validationMetric) == 0 {
		metricName = trainingMetric
		metricType = TRAINING
	}
	if len(trainingMetric) == 0 && len(validationMetric) > 0 {
		metricName = validationMetric
		metricType = VALIDATION
	}
	if len(metricName) == 0 {
		return nil, endTime,
			errors.New("must provide one training metric, or one validation metric, but not both")
	}

	type batchesWrapper struct {
		Batches int32     `db:"batches_processed"`
		EndTime time.Time `db:"end_time"`
	}

	var query string
	if metricType == TRAINING {
		query = `
SELECT (s.prior_batches_processed + num_batches) AS batches_processed,
  max(s.end_time) as end_time
FROM trials t INNER JOIN steps s ON t.id=s.trial_id
WHERE t.experiment_id=$1
  AND s.state = 'COMPLETED'
  AND s.metrics->'avg_metrics' ? $2
  AND s.end_time > $3
GROUP BY batches_processed;`
	} else {
		query = `
SELECT (s.prior_batches_processed + num_batches) AS batches_processed
  max(v.end_time) as end_time
FROM trials t INNER JOIN steps s ON t.id=s.trial_id
  LEFT OUTER JOIN validations v ON s.id=v.step_id AND s.trial_id=v.trial_id
WHERE t.experiment_id=$1
  AND v.state = 'COMPLETED'
  AND v.metrics->'validation_metrics' ? $2
  AND v.end_time > $3
GROUP BY batches_processed;`
	}

	rows, err := db.sql.Queryx(query, experimentID, metricName, startTime)
	if err != nil {
		return nil, endTime, errors.Wrapf(err,
			"failed to get metric batches for experiment %d and %s metric %s",
			experimentID, metricType, metricName)
	}
	defer rows.Close()

	for rows.Next() {
		var row batchesWrapper
		err = rows.StructScan(&row)
		if err != nil {
			return nil, endTime, errors.Wrapf(err,
				"error scanning training metric names for experiment %d",
				experimentID)
		}
		batches = append(batches, row.Batches)
		endTime = row.EndTime
	}

	return batches, endTime, nil
}
