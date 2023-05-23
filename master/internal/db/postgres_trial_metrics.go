package db

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

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

/*
rollbackMetrics ensures old training and validation metrics from a previous run id are archived.
*/
func rollbackMetrics(ctx context.Context, tx *sqlx.Tx, runID, trialID,
	lastProcessedBatch int32, pType MetricPartitionType,
) (int, error) {
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

func (db *PgDB) addRawMetrics(ctx context.Context, tx *sqlx.Tx, metricsBody *map[string]interface{},
	runID, trialID, lastProcessedBatch int32,
	pType MetricPartitionType, mType *string,
) (int, error) {
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

func customMetricTypeToPartitionType(mType string) MetricPartitionType {
	// TODO(hamid): remove partition_type once we move away from pg10 and
	// we can use DEFAULT partitioning.
	switch mType {
	case model.TrainingMetricType.ToString():
		return TrainingMetric
	case model.ValidationMetricType.ToString():
		return ValidationMetric
	default:
		return GenericMetric
	}
}

// AddTrainingMetrics [DEPRECATED] adds a completed step to the database with the given training
// metrics. If these training metrics occur before any others, a rollback is assumed and later
// training and validation metrics are cleaned up.
func (db *PgDB) AddTrainingMetrics(ctx context.Context, m *trialv1.TrialMetrics) error {
	_, err := db.addTrialMetrics(ctx, m, TrainingMetric, nil)
	return err
}

// AddValidationMetrics [DEPRECATED] adds a completed validation to the database with the given
// validation metrics. If these validation metrics occur before any others, a rollback
// is assumed and later metrics are cleaned up from the database.
func (db *PgDB) AddValidationMetrics(
	ctx context.Context, m *trialv1.TrialMetrics,
) error {
	_, err := db.addTrialMetrics(ctx, m, ValidationMetric, nil)
	return err
}

// AddTrialMetrics persists the given trial metrics to the database.
func (db *PgDB) AddTrialMetrics(
	ctx context.Context, m *trialv1.TrialMetrics, mType string,
) error {
	pType := customMetricTypeToPartitionType(mType)
	_, err := db.addTrialMetrics(ctx, m, pType, &mType)
	return err
}

// GetMetrics returns a subset metrics of the requested type for the given trial ID.
func GetMetrics(ctx context.Context, trialID, afterBatches, limit int,
	mType string,
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
		query.Where("custom_type = ?", &mType)
	}

	err := query.
		Order("trial_id", "trial_run_id", "total_batches").
		Limit(limit).
		Scan(ctx, &res)

	return res, err
}
