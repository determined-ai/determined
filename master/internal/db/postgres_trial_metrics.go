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

type metricsBody struct {
	BatchMetrics interface{}
	AvgMetrics   *structpb.Struct
	isValidation bool
}

func (b metricsBody) ToJSONObj() *model.JSONObj {
	// we should probably move to avoid special casing based on metric type here.
	metricsJSONPath := model.TrialMetricsJSONPath(b.isValidation)
	body := model.JSONObj{
		metricsJSONPath: b.AvgMetrics,
	}

	if b.isValidation {
		return &body
	}

	body["batch_metrics"] = b.BatchMetrics
	return &body
}

func (b *metricsBody) LoadJSON(body *model.JSONObj) (err error) {
	metricsJSONPath := model.TrialMetricsJSONPath(b.isValidation)

	avgMetricsVal, exists := (*body)[metricsJSONPath]
	if !exists {
		return fmt.Errorf("expected key %s in JSON body", metricsJSONPath)
	}
	avgMetrics, ok := avgMetricsVal.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to deserialize %s", metricsJSONPath)
	}
	if b.AvgMetrics, err = structpb.NewStruct(avgMetrics); err != nil {
		return errors.Wrapf(err, "failed to convert avg metrics to structpb.Struct")
	}

	if b.isValidation {
		return nil
	}

	batchMetricsVal, exists := (*body)["batch_metrics"]
	if exists {
		b.BatchMetrics = batchMetricsVal
	}

	return nil
}

func newMetricsBody(
	avgMetrics *structpb.Struct,
	batchMetrics []*structpb.Struct,
	isValidation bool,
) *metricsBody {
	var bMetrics any = nil
	if len(batchMetrics) != 0 {
		bMetrics = batchMetrics
	}
	return &metricsBody{
		AvgMetrics:   avgMetrics,
		BatchMetrics: bMetrics,
		isValidation: isValidation,
	}
}

// BunSelectMetricsQuery sets up a bun select query for based on new metrics table
// simplifying some weirdness we set up for pg10 support.
func BunSelectMetricsQuery(metricGroup model.MetricGroup, inclArchived bool) *bun.SelectQuery {
	pType := customMetricGroupToPartitionType(metricGroup)
	q := Bun().NewSelect().
		Where("partition_type = ?", pType).
		Where("archived = ?", inclArchived)
	if pType == GenericMetric {
		q.Where("metric_group = ?", metricGroup)
	}
	return q
}

// BunSelectMetricGroupNames sets up a bun select query for getting all the metric group and names.
func BunSelectMetricGroupNames() *bun.SelectQuery {
	return Bun().NewSelect().Table("trials").
		ColumnExpr("jsonb_object_keys(summary_metrics) as json_path").
		ColumnExpr("jsonb_object_keys(summary_metrics->jsonb_object_keys(summary_metrics))" +
			" as metric_name").
		Where("summary_metrics IS NOT NULL").
		Order("json_path").Order("metric_name")
}

/*
rollbackMetrics ensures old training and validation metrics from a previous run id are archived.
*/
func rollbackMetrics(ctx context.Context, tx *sqlx.Tx, runID, trialID,
	lastProcessedBatch int32, mGroup model.MetricGroup,
) (int, error) {
	pType := customMetricGroupToPartitionType(mGroup)
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

// addMetricsWithMerge inserts a set of metrics to the database allowing for metric merges.
func (db *PgDB) addMetricsWithMerge(ctx context.Context, tx *sqlx.Tx, mBody *metricsBody,
	runID, trialID, lastProcessedBatch int32, mGroup model.MetricGroup,
) (metricID int, addedMetrics *metricsBody, err error) {
	var existingBodyJSON model.JSONObj
	err = tx.QueryRowContext(ctx, `
SELECT COALESCE((SELECT metrics FROM metrics
WHERE archived = false
AND total_batches = $1
AND trial_id = $2
AND partition_type = $3
AND metric_group = $4), NULL)
FOR UPDATE`,
		lastProcessedBatch, trialID,
		customMetricGroupToPartitionType(mGroup), mGroup).Scan(&existingBodyJSON)
	if err != nil {
		return 0, nil, errors.Wrap(err, "getting old metrics")
	}
	needsMerge := existingBodyJSON != nil

	if !needsMerge {
		id, err := db.addRawMetrics(ctx, tx, mBody, runID, trialID, lastProcessedBatch, mGroup)
		return id, mBody, err
	}

	existingBody := &metricsBody{isValidation: mGroup == model.ValidationMetricGroup}
	if err = existingBody.LoadJSON(&existingBodyJSON); err != nil {
		return 0, nil, err
	}
	finalBody, err := shallowUnionMetrics(existingBody, mBody)
	if err != nil {
		return 0, nil, err
	}
	id, err := db.updateRawMetrics(ctx, tx, finalBody, runID, trialID, lastProcessedBatch, mGroup)
	return id, mBody, err
}

func (db *PgDB) updateRawMetrics(ctx context.Context, tx *sqlx.Tx, mBody *metricsBody,
	runID, trialID, lastProcessedBatch int32, mGroup model.MetricGroup,
) (int, error) {
	if err := mGroup.Validate(); err != nil {
		return 0, err
	}
	pType := customMetricGroupToPartitionType(mGroup)

	var metricRowID int
	//nolint:execinquery // we want to get the id.
	if err := tx.QueryRowContext(ctx, `
UPDATE metrics
SET metrics = $1
WHERE archived = false
AND trial_id = $2
AND partition_type = $3
AND metric_group = $4
AND total_batches = $5
RETURNING id`,
		*mBody.ToJSONObj(), trialID, pType, mGroup, lastProcessedBatch,
	).Scan(&metricRowID); err != nil {
		return 0, errors.Wrap(err, "updating metrics")
	}

	return metricRowID, nil
}

// addRawMetrics inserts a set of raw metrics to the database and returns the metric id.
func (db *PgDB) addRawMetrics(ctx context.Context, tx *sqlx.Tx, mBody *metricsBody,
	runID, trialID, lastProcessedBatch int32, mGroup model.MetricGroup,
) (int, error) {
	if err := mGroup.Validate(); err != nil {
		return 0, err
	}
	pType := customMetricGroupToPartitionType(mGroup)

	var metricRowID int
	// ON CONFLICT clause is not supported with partitioned tables (SQLSTATE 0A000)
	//nolint:execinquery // we want to get the id.
	if err := tx.QueryRowContext(ctx, `
INSERT INTO metrics
	(trial_id, trial_run_id, end_time, metrics, total_batches, partition_type, metric_group)
VALUES
	($1, $2, now(), $3, $4, $5, $6)
RETURNING id`,
		trialID, runID, *mBody.ToJSONObj(), lastProcessedBatch, pType, mGroup,
	).Scan(&metricRowID); err != nil {
		return metricRowID, errors.Wrap(err, "inserting metrics")
	}

	return metricRowID, nil
}

func customMetricGroupToPartitionType(mGroup model.MetricGroup) MetricPartitionType {
	// TODO(hamid): remove partition_type once we move away from pg10 and
	// we can use DEFAULT partitioning.
	switch mGroup {
	case model.TrainingMetricGroup:
		return TrainingMetric
	case model.ValidationMetricGroup:
		return ValidationMetric
	default:
		return GenericMetric
	}
}

// AddTrainingMetrics [DEPRECATED] adds a completed step to the database with the given training
// metrics. If these training metrics occur before any others, a rollback is assumed and later
// training and validation metrics are cleaned up.
func (db *PgDB) AddTrainingMetrics(ctx context.Context, m *trialv1.TrialMetrics) error {
	_, err := db.addTrialMetrics(ctx, m, model.TrainingMetricGroup)
	return err
}

// AddValidationMetrics [DEPRECATED] adds a completed validation to the database with the given
// validation metrics. If these validation metrics occur before any others, a rollback
// is assumed and later metrics are cleaned up from the database.
func (db *PgDB) AddValidationMetrics(
	ctx context.Context, m *trialv1.TrialMetrics,
) error {
	_, err := db.addTrialMetrics(ctx, m, model.ValidationMetricGroup)
	return err
}

// AddTrialMetrics persists the given trial metrics to the database.
func (db *PgDB) AddTrialMetrics(
	ctx context.Context, m *trialv1.TrialMetrics, mGroup model.MetricGroup,
) error {
	_, err := db.addTrialMetrics(ctx, m, mGroup)
	return err
}

// GetMetrics returns a subset metrics of the requested type for the given trial ID.
func GetMetrics(ctx context.Context, trialID, afterBatches, limit int,
	mGroup model.MetricGroup,
) ([]*trialv1.MetricsReport, error) {
	var res []*trialv1.MetricsReport
	pType := customMetricGroupToPartitionType(mGroup)
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
		query.Where("metric_group = ?", mGroup)
	}

	err := query.
		Order("trial_id", "trial_run_id", "total_batches").
		Limit(limit).
		Scan(ctx, &res)

	return res, err
}

// shallowUnionMetrics unions non-overlapping keys of two metrics bodies.
func shallowUnionMetrics(oldBody, newBody *metricsBody) (*metricsBody, error) {
	if oldBody == nil {
		return newBody, nil
	}

	if newBody == nil {
		return oldBody, nil
	}

	// disallow batch metrics from being overwritten.
	if oldBody.BatchMetrics != nil && newBody.BatchMetrics != nil {
		return nil, fmt.Errorf("overwriting batch metrics is not supported")
	}

	oldAvgMetrics := oldBody.AvgMetrics
	newAvgMetrics := newBody.AvgMetrics
	for key, newValue := range newAvgMetrics.GetFields() {
		// we cannot calculate min/max efficiently for replaced metric values
		// so we disallow it.
		if _, ok := oldAvgMetrics.GetFields()[key]; ok {
			return nil, fmt.Errorf("overwriting existing metric keys is not supported,"+
				" conflicting key: %s", key)
		}
		oldAvgMetrics.GetFields()[key] = newValue
	}

	oldBody.AvgMetrics = oldAvgMetrics

	return oldBody, nil
}
