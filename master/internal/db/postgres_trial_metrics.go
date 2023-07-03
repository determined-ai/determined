package db

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// MetricPartitionType denotes what type the metric is. This is planned to be deprecated
// once we upgrade to pg11 and can use DEFAULT partitioning.
type MetricPartitionType string

const (
	// TrainingMetric designates metrics from training steps.
	TrainingMetric MetricPartitionType = "TRAINING"
	// ValidationMetric designates metrics from validation steps.
	ValidationMetric MetricPartitionType = "VALIDATION"
	// GenericMetric designates metrics from other sources.
	GenericMetric MetricPartitionType = "GENERIC"
)

// BunSelectMetricsQuery sets up a bun select query for based on new metrics table
// simplifying some weirdness we set up for pg10 support.
func BunSelectMetricsQuery(metricType model.MetricType, inclArchived bool) *bun.SelectQuery {
	pType := customMetricTypeToPartitionType(metricType)
	q := Bun().NewSelect().
		Where("partition_type = ?", pType).
		Where("archived = ?", inclArchived)
	if pType == GenericMetric {
		q.Where("custom_type = ?", metricType)
	}
	return q
}

// BunSelectMetricTypeNames sets up a bun select query for getting all the metric type and names.
func BunSelectMetricTypeNames() *bun.SelectQuery {
	return Bun().NewSelect().Table("trials").
		ColumnExpr("DISTINCT jsonb_object_keys(summary_metrics) as json_path").
		ColumnExpr("jsonb_object_keys(summary_metrics->jsonb_object_keys(summary_metrics))" +
			" as metric_name").
		Where("summary_metrics IS NOT NULL").
		Order("json_path").Order("metric_name")
}

/*
rollbackMetrics ensures old training and validation metrics from a previous run id are archived.
*/
func rollbackMetrics(ctx context.Context, tx *sqlx.Tx, runID, trialID,
	lastProcessedBatch int32, mType model.MetricType,
) (int, error) {
	pType := customMetricTypeToPartitionType(mType)
	res, err := tx.ExecContext(ctx, `
UPDATE metrics SET archived = true
WHERE trial_id = $1
  AND archived = false
  AND trial_run_id < $2
	-- we mark metrics reported in the same table with the same batch number
	-- as the metric being added as "archived"
  AND (
		(
			partition_type != $4 AND total_batches > $3
		) OR
		(
			partition_type = $4 AND total_batches >= $3
		)
		
	);
	`, trialID, runID, lastProcessedBatch, pType)
	if err != nil {
		return 0, errors.Wrap(err, "archiving metrics")
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "checking for metric rollbacks")
	}
	return int(affectedRows), nil
}

// addMetricsWithMerge inserts a set of metrics to the database and returns the metric id,
// the new metrics, and the previous metrics in case of a merge.
func (db *PgDB) addMetricsWithMerge(ctx context.Context, tx *sqlx.Tx, metricsBody *model.JSONObj,
	runID, trialID, lastProcessedBatch int32, mType model.MetricType,
) (int, *model.JSONObj, *model.JSONObj, error) {
	// TODO: fetch previous metrics
	// TODO: merge previous metrics with new metrics
	// TODO: addRawMetrics new metrics
	newMetricsBody := *metricsBody
	id, err := db.addRawMetrics(ctx, tx, &newMetricsBody, runID, trialID, lastProcessedBatch, mType)

	return id, &newMetricsBody, metricsBody, err
}

// addRawMetrics inserts a set of raw metrics to the database and returns the metric id.
func (db *PgDB) addRawMetrics(ctx context.Context, tx *sqlx.Tx, metricsBody *model.JSONObj,
	runID, trialID, lastProcessedBatch int32, mType model.MetricType,
) (int, error) {
	pType := customMetricTypeToPartitionType(mType)

	if err := mType.Validate(); err != nil {
		return 0, err
	}

	var metricRowID int
	//nolint:execinquery // we want to get the id.
	if err := tx.QueryRowContext(ctx, `
INSERT INTO metrics
	(trial_id, trial_run_id, end_time, metrics, total_batches, partition_type, custom_type)
VALUES
	($1, $2, now(), $3, $4, $5, $6)
RETURNING id`,
		trialID, runID, *metricsBody, lastProcessedBatch, pType, mType,
	).Scan(&metricRowID); err != nil {
		return metricRowID, errors.Wrap(err, "inserting metrics")
	}

	return metricRowID, nil
}

func customMetricTypeToPartitionType(mType model.MetricType) MetricPartitionType {
	// TODO(hamid): remove partition_type once we move away from pg10 and
	// we can use DEFAULT partitioning.
	switch mType {
	case model.TrainingMetricType:
		return TrainingMetric
	case model.ValidationMetricType:
		return ValidationMetric
	default:
		return GenericMetric
	}
}

// addTrialMetrics inserts a set of trial metrics to the database.
func (db *PgDB) addTrialMetrics(
	ctx context.Context, m *trialv1.TrialMetrics, mType model.MetricType,
) (rollbacks int, err error) {
	switch v := m.Metrics.AvgMetrics.Fields["epoch"].AsInterface().(type) {
	case float64, nil:
	default:
		return 0, fmt.Errorf("cannot add metric with non numeric 'epoch' value got %v", v)
	}
	return rollbacks, db.withTransaction(fmt.Sprintf("add trial metrics %s", mType),
		func(tx *sqlx.Tx) error {
			rollbacks, err = db._addTrialMetricsTx(ctx, tx, m, mType)
			return err
		})
}

// AddTrainingMetrics [DEPRECATED] adds a completed step to the database with the given training
// metrics. If these training metrics occur before any others, a rollback is assumed and later
// training and validation metrics are cleaned up.
func (db *PgDB) AddTrainingMetrics(ctx context.Context, m *trialv1.TrialMetrics) error {
	_, err := db.addTrialMetrics(ctx, m, model.TrainingMetricType)
	return err
}

// AddValidationMetrics [DEPRECATED] adds a completed validation to the database with the given
// validation metrics. If these validation metrics occur before any others, a rollback
// is assumed and later metrics are cleaned up from the database.
func (db *PgDB) AddValidationMetrics(
	ctx context.Context, m *trialv1.TrialMetrics,
) error {
	_, err := db.addTrialMetrics(ctx, m, model.ValidationMetricType)
	return err
}

// AddTrialMetrics persists the given trial metrics to the database.
func (db *PgDB) AddTrialMetrics(
	ctx context.Context, m *trialv1.TrialMetrics, mType model.MetricType,
) error {
	_, err := db.addTrialMetrics(ctx, m, mType)
	return err
}

// GetMetrics returns a subset metrics of the requested type for the given trial ID.
func GetMetrics(ctx context.Context, trialID, afterBatches, limit int,
	mType model.MetricType,
) ([]*trialv1.MetricsReport, error) {
	var res []*trialv1.MetricsReport
	pType := customMetricTypeToPartitionType(mType)
	query := Bun().NewSelect().Table("metrics").
		Column("trial_id", "metrics", "total_batches", "archived", "id", "trial_run_id").
		ColumnExpr("proto_time(end_time) AS end_time").
		Where("partition_type = ?", pType).
		Where("trial_id = ?", trialID).
		Where("total_batches > ?", afterBatches).
		Where("archived = false")

	if pType == GenericMetric {
		// Going off of our current schema were looking for custom types in our legacy
		// metrics tables is pointless.
		query.Where("custom_type = ?", mType)
	}

	err := query.
		Order("trial_id", "trial_run_id", "total_batches").
		Limit(limit).
		Scan(ctx, &res)

	return res, err
}

// removeMetricsFromSummary removes metrics from the summary that is already
// computed.
func removeMetricsFromSummary(
	summaryMetrics model.JSONObj, metrics *structpb.Struct,
) model.JSONObj {
	// TODO
	return summaryMetrics
}

func removeDuplicateEntry(ctx context.Context, tx *sqlx.Tx,
	m *trialv1.TrialMetrics, mType model.MetricType,
) (bool, error) {
	_, err := tx.ExecContext(ctx, `
DELETE FROM metrics
WHERE archived = false
AND total_batches = $1
AND trial_id = $2
AND partition_type = $3
AND custom_type = $4
`, m.StepsCompleted, m.TrialId, customMetricTypeToPartitionType(mType), mType)
	if err != nil {
		return false, errors.Wrap(err, "updating duplicate metric")
	}
	// TODO. return deleted metric if any.
	// TODO update summary.
	return false, nil
}

func mergeMetrics(oldBody, newBody *model.JSONObj) *model.JSONObj {
	// TODO
	// can this be done in db? should it?
	return newBody
}
