package db

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

/*
rollbackMetrics ensures old training and validation metrics from a previous run id are archived.
DISCUSS: how do we decide if a utility should be namespaced under PgDB or not?
the goal is to make it clear we don't have direct db access other than through tx.
*/
func rollbackMetrics(ctx context.Context, tx *sqlx.Tx, runID, trialID,
	lastProcessedBatch int32, isValidation bool,
) (int, error) {
	pType := model.TrainingMetric
	if isValidation {
		pType = model.ValidationMetric
	}

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
	pType model.MetricPartitionType, mType *string,
) (int, error) {
	var metricRowID int
	if err := tx.QueryRowContext(ctx, `
INSERT INTO metrics
	(trial_id, trial_run_id, end_time, metrics, total_batches, partition_type, custom_type)
VALUES
	($1, $2, now(), $3, $4, $5, $6)
RETURNING id`,
		trialID, runID, *metricsBody, lastProcessedBatch, pType, mType, // CHECK: are nulls handled?
	).Scan(&metricRowID); err != nil {
		return metricRowID, errors.Wrap(err, "inserting metrics")
	}

	return metricRowID, nil
}

func customMetricTypeToPartitionType(mType string) model.MetricPartitionType {
	// TODO(hamid): remove partition_type once we move away from pg10 and
	// we can use DEFAULT partitioning.
	switch mType {
	case string(model.TrainingMetric): // FIXME: case sensitive.
		return model.TrainingMetric
	case string(model.ValidationMetric):
		return model.ValidationMetric
	default:
		return model.GenericMetric
	}
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
	// TODO view on top of metrics table?
	return res, Bun().NewSelect().Table("metrics").
		Column("trial_id", "metrics", "total_batches", "archived", "id", "trial_run_id").
		ColumnExpr("proto_time(end_time) AS end_time").
		Where("trial_id = ?", trialID).
		Where("total_batches > ?", afterBatches).
		Where("archived = false").
		Where("partition_type = ?", customMetricTypeToPartitionType(mType)).
		Order("trial_id", "trial_run_id", "total_batches").
		Limit(limit).
		Scan(ctx, &res)
}
