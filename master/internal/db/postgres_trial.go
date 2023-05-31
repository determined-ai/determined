package db

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// TODO: rename raw_steps.

// AddTrial adds the trial to the database and sets its ID.
func (db *PgDB) AddTrial(trial *model.Trial) error {
	if trial.ID != 0 {
		return errors.Errorf("error adding a trial with non-zero id %v", trial.ID)
	}

	if err := db.namedGet(&trial.ID, `
INSERT INTO trials
(task_id, request_id, experiment_id, state, start_time, end_time,
hparams, warm_start_checkpoint_id, seed)
VALUES (:task_id, :request_id, :experiment_id, :state, :start_time,
	:end_time, :hparams, :warm_start_checkpoint_id, :seed)
RETURNING id`, trial); err != nil {
		// Assume the foreign key constraint is handled by the database.
		return errors.Wrapf(err, "error inserting trial %v", *trial)
	}

	return nil
}

// TrialByID looks up a trial by ID, returning an error if none exists.
func (db *PgDB) TrialByID(id int) (*model.Trial, error) {
	var trial model.Trial
	err := db.query(`
SELECT id, COALESCE(task_id, '') AS task_id, request_id, experiment_id, state, start_time,
	end_time, hparams, warm_start_checkpoint_id, seed, total_batches
FROM trials
WHERE id = $1`, &trial, id)
	return &trial, errors.Wrapf(err, "error querying for trial %v", id)
}

// TrialByExperimentAndRequestID looks up a trial, returning an error if none exists.
func (db *PgDB) TrialByExperimentAndRequestID(
	experimentID int, requestID model.RequestID,
) (*model.Trial, error) {
	var trial model.Trial
	err := db.query(`
SELECT id, task_id, request_id, experiment_id, state, start_time,
  end_time, hparams, warm_start_checkpoint_id, seed, total_batches
FROM trials
WHERE experiment_id = $1 AND request_id = $2`, &trial, experimentID, requestID)
	return &trial, errors.Wrapf(err, "error querying for trial %v", requestID)
}

// UpdateTrial updates an existing trial. Fields that are nil or zero are not
// updated.  end_time is set if the trial moves to a terminal state.
func (db *PgDB) UpdateTrial(id int, newState model.State) error {
	trial, err := db.TrialByID(id)
	if err != nil {
		return errors.Wrapf(err, "error finding trial %v to update", id)
	}

	if trial.State == newState {
		return nil
	}

	if !model.TrialTransitions[trial.State][newState] {
		return errors.Errorf("illegal transition %v -> %v for trial %v",
			trial.State, newState, trial.ID)
	}
	toUpdate := []string{"state"}
	trial.State = newState
	if model.TerminalStates[newState] {
		now := time.Now().UTC()
		trial.EndTime = &now
		toUpdate = append(toUpdate, "end_time")
	}

	return db.withTransaction("update_trial", func(tx *sqlx.Tx) error {
		// Only the trial actor updates this row, and it does so in a serialized
		// fashion already, so this transaction is more a matter of atomicity.
		if err := namedExecOne(tx, fmt.Sprintf(`
UPDATE trials
%v
WHERE id = :id`, setClause(toUpdate)), trial); err != nil {
			return errors.Wrapf(err, "error updating (%v) in trial %v",
				strings.Join(toUpdate, ", "), id)
		}

		if model.TerminalStates[newState] && trial.EndTime != nil {
			return completeTask(tx, trial.TaskID, *trial.EndTime)
		}

		return nil
	})
}

// UpdateTrialRunnerState updates a trial runner's state.
func (db *PgDB) UpdateTrialRunnerState(id int, state string) error {
	return db.UpdateTrialRunnerMetadata(id, &trialv1.TrialRunnerMetadata{State: state})
}

// UpdateTrialRunnerMetadata updates a trial's metadata about its runner.
func (db *PgDB) UpdateTrialRunnerMetadata(id int, md *trialv1.TrialRunnerMetadata) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET runner_state = $2
WHERE id = $1`, id, md.State); err != nil {
		return errors.Wrap(err, "saving trial runner state")
	}
	return nil
}

// TrialRunIDAndRestarts returns the run id and restart count for a trial.
func (db *PgDB) TrialRunIDAndRestarts(trialID int) (int, int, error) {
	var runID, restart int
	if err := db.sql.QueryRowx(`
SELECT run_id, restarts
FROM trials
WHERE id = $1`, trialID).Scan(&runID, &restart); err != nil {
		return 0, 0, errors.Wrap(err, "failed to scan trial restart count")
	}
	return runID, restart, nil
}

// UpdateTrialRunID sets the trial's run ID.
func (db *PgDB) UpdateTrialRunID(id, runID int) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET run_id = $2
WHERE id = $1`, id, runID); err != nil {
		return errors.Wrap(err, "updating trial run id")
	}
	return nil
}

// UpdateTrialRestarts sets the trial's restart count.
func (db *PgDB) UpdateTrialRestarts(id, restartCount int) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET restarts = $2
WHERE id = $1`, id, restartCount); err != nil {
		return errors.Wrap(err, "updating trial restarts")
	}
	return nil
}

// fullTrialSummaryMetricsRecompute recomputes all summary metrics for a given trial.
func (db *PgDB) fullTrialSummaryMetricsRecompute(
	ctx context.Context, tx *sqlx.Tx, trialID int,
) error {
	trainSummary, err := db.calculateFullTrialSummaryMetrics(
		ctx, tx, trialID, "avg_metrics", TrainingMetric)
	if err != nil {
		return fmt.Errorf("rollback computing training summary metrics: %w", err)
	}
	valSummary, err := db.calculateFullTrialSummaryMetrics(
		ctx, tx, trialID, "validation_metrics", ValidationMetric)
	if err != nil {
		return fmt.Errorf("rollback computing validation summary metrics: %w", err)
	}

	updatedSummaryMetrics := model.JSONObj{}
	if len(trainSummary) > 0 {
		updatedSummaryMetrics["avg_metrics"] = trainSummary
	}
	if len(valSummary) > 0 {
		updatedSummaryMetrics["validation_metrics"] = valSummary
	}

	if _, err := tx.ExecContext(ctx, `UPDATE trials SET summary_metrics = $1,
	summary_metrics_timestamp = NOW() WHERE id = $2`, updatedSummaryMetrics, trialID); err != nil {
		return fmt.Errorf("rollback updating trial summary metrics: %w", err)
	}
	return nil
}

func (db *PgDB) calculateFullTrialSummaryMetrics(
	ctx context.Context, tx *sqlx.Tx, trialID int, jsonPath string, partition MetricPartitionType,
) (model.JSONObj, error) {
	//nolint: execinquery
	rows, err := tx.QueryContext(ctx, `
-- Returns pairs of metric names and trial_ids and if they are numeric or not.
WITH trial_metrics as (
SELECT
	name,
	trial_id,
	CASE sum(entries)
		WHEN sum(entries) FILTER (WHERE metric_type = 'number') THEN 'number'
		WHEN sum(entries) FILTER (WHERE metric_type = 'string') THEN 'string'
		WHEN sum(entries) FILTER (WHERE metric_type = 'date') THEN 'date'
		WHEN sum(entries) FILTER (WHERE metric_type = 'object') THEN 'object'
		WHEN sum(entries) FILTER (WHERE metric_type = 'boolean') THEN 'boolean'
		WHEN sum(entries) FILTER (WHERE metric_type = 'array') THEN 'array'
		WHEN sum(entries) FILTER (WHERE metric_type = 'null') THEN 'null'
		ELSE 'string'
	END as metric_type
FROM (
	SELECT
	name,
	CASE
		WHEN jsonb_typeof(metrics->$2->name) = 'string' THEN
			CASE
				WHEN (metrics->$2->name)::text = '"Infinity"'::text THEN 'number'
				WHEN (metrics->$2->name)::text = '"-Infinity"'::text THEN 'number'
				WHEN (metrics->$2->name)::text = '"NaN"'::text THEN 'number'
				WHEN metrics->$2->>name ~
					'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$' THEN 'date'
				ELSE 'string'
			END
		ELSE jsonb_typeof(metrics->$2->name)
	END as metric_type,
	trial_id,
	count(1) as entries
	FROM (
		SELECT DISTINCT
		jsonb_object_keys(s.metrics->$2) as name
		FROM metrics s
		WHERE s.trial_id = $1 AND partition_type = $3 AND not archived
	) names, metrics
	JOIN trials ON trial_id = trials.id
	WHERE trials.id = $1 AND metrics.partition_type = $3 AND not metrics.archived
	GROUP BY name, metric_type, trial_id
) typed
where metric_type IS NOT NULL
GROUP BY name, trial_id
ORDER BY trial_id, name
),
-- Filters to only numeric metrics.
numeric_trial_metrics as (
SELECT name, trial_id
FROM trial_metrics
WHERE metric_type = 'number'
),
-- Calculates count, sum, min, max on each numeric metric name and trial ID pair.
-- Also adds just the name for non numeric metrics to ensure we record every metric.
trial_metric_aggs as (
SELECT
	name,
	ntm.trial_id,
	count(1) as count_agg,
	sum((metrics.metrics->$2->>name)::double precision) as sum_agg,
	min((metrics.metrics->$2->>name)::double precision) as min_agg,
	max((metrics.metrics->$2->>name)::double precision) as max_agg,
	'number' as metric_type
FROM numeric_trial_metrics ntm INNER JOIN metrics
ON metrics.trial_id=ntm.trial_id
WHERE metrics.metrics->$2->name IS NOT NULL AND metrics.partition_type = $3 AND not metrics.archived
GROUP BY 1, 2
UNION
SELECT
	name,
	trial_id,
	NULL as count_agg,
	NULL as sum,
	NULL as min,
	NULL as max,
	metric_type as metric_type
FROM trial_metrics
WHERE metric_type != 'number'
),
-- Gets the last reported metric for each trial. Note if we report
-- {"a": 1} and {"b": 1} we consider {"b": 1} to be the last reported
-- metric and "a"'s last will be NULL.
latest_metrics as (
  SELECT s.trial_id,
	unpacked.key as name,
	unpacked.value as latest_value
  FROM (
	  SELECT s.*,
		ROW_NUMBER() OVER(
		  PARTITION BY s.trial_id
		  ORDER BY s.end_time DESC
		) as rank
	  FROM metrics s
	  JOIN trials ON s.trial_id = trials.id
	  WHERE s.trial_id = $1 AND partition_type = $3 AND not archived
	) s, jsonb_each(s.metrics->$2) unpacked
  WHERE s.rank = 1
),
-- Adds the last reported metric to the aggregation.
combined_latest_agg as (SELECT
	coalesce(lt.trial_id, tma.trial_id) as trial_id,
	coalesce(lt.name, tma.name) as name,
	tma.count_agg,
	tma.sum_agg,
	tma.min_agg,
	tma.max_agg,
	lt.latest_value,
	tma.metric_type
FROM latest_metrics lt FULL OUTER JOIN trial_metric_aggs tma ON
	lt.trial_id = tma.trial_id AND lt.name = tma.name
) SELECT name, jsonb_build_object(
    'count', count_agg,
    'sum', sum_agg,
    'min', CASE WHEN max_agg = 'NaN'::double precision
        THEN 'NaN'::double precision ELSE min_agg END,
    'max', max_agg,
    'last', latest_value,
    'type', metric_type
) FROM combined_latest_agg;
`, trialID, jsonPath, partition)
	if err != nil {
		return nil, errors.Wrapf(err, "getting full compute trial %d summary metrics", trialID)
	}

	metrics := model.JSONObj{}
	defer rows.Close()
	for rows.Next() {
		var metric model.JSONObj
		var name string
		if err = rows.Scan(&name, &metric); err != nil {
			return nil, fmt.Errorf("scanning summary metric row: %w", err)
		}
		metrics[name] = metric
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err summary metric full compute: %w", err)
	}

	return metrics, nil
}

// updateLatestValidationID updates latest validation based on validations table.
func (db *PgDB) updateLatestValidationID(ctx context.Context, tx *sqlx.Tx, trialID int) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE trials SET latest_validation_id = (
			SELECT validations.id
			FROM validations
			JOIN trials t ON validations.trial_id = t.id
			JOIN experiments e on t.experiment_id = e.id
			WHERE trial_id = $1 AND (
				validations.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric')
			) IS NOT NULL
			ORDER BY validations.end_time DESC
			LIMIT 1
		) WHERE id = $1`, trialID); err != nil {
		return fmt.Errorf("updating latest validation id for trial %d: %w", trialID, err)
	}
	return nil
}

// updateTotalBatches update precomputed total_batches based on existing steps and validations.
func (db *PgDB) updateTotalBatches(ctx context.Context, tx *sqlx.Tx, trialID int) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE trials t
		SET total_batches = COALESCE(latest.total_batches_processed, 0)
		FROM (
				SELECT max(m.total_batches) AS total_batches_processed
				FROM metrics m
				WHERE m.trial_id = $1 AND m.archived = false
			) AS latest
		WHERE t.id = $1
		`, trialID); err != nil {
		return errors.Wrap(err, "error computing total_batches")
	}
	return nil
}

func (db *PgDB) _addTrialMetricsTx(
	ctx context.Context, tx *sqlx.Tx, m *trialv1.TrialMetrics, pType MetricPartitionType,
	mType *string,
) (rollbacks int, err error) {
	isValidation := pType == ValidationMetric

	metricsJSONPath := "avg_metrics"
	metricsBody := map[string]interface{}{
		metricsJSONPath: m.Metrics.AvgMetrics,
		"batch_metrics": m.Metrics.BatchMetrics,
	}
	if isValidation {
		metricsJSONPath = "validation_metrics"
		metricsBody = map[string]interface{}{
			metricsJSONPath: m.Metrics.AvgMetrics,
		}
	}

	if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
		return rollbacks, err
	}

	if rollbacks, err = rollbackMetrics(ctx, tx, m.TrialRunId, m.TrialId, m.StepsCompleted,
		pType); err != nil {
		return rollbacks, err
	}
	var summaryMetrics model.JSONObj
	err = tx.QueryRowContext(ctx, `
		SELECT summary_metrics FROM trials WHERE id = $1 FOR UPDATE;
	`, m.TrialId).Scan(&summaryMetrics)
	if err != nil {
		return rollbacks, fmt.Errorf("error getting summary metrics from trials: %w", err)
	}

	metricRowID, err := db.addRawMetrics(ctx, tx, &metricsBody, m.TrialRunId,
		m.TrialId, m.StepsCompleted, pType, mType)
	if err != nil {
		return rollbacks, err
	}

	switch {
	case rollbacks != 0:
		if err := db.updateTotalBatches(ctx, tx, int(m.TrialId)); err != nil {
			return rollbacks, errors.Wrap(err, "rollback")
		}

		if err := db.updateLatestValidationID(ctx, tx, int(m.TrialId)); err != nil {
			return rollbacks, fmt.Errorf(
				"rollback updating latest validation ID for trial %d: %w", m.TrialId, err)
		}

		if err := db.fullTrialSummaryMetricsRecompute(ctx, tx, int(m.TrialId)); err != nil {
			return rollbacks, errors.Wrap(err, "error on rollback compute of summary metrics")
		}
	case pType == GenericMetric:
		if _, err := tx.ExecContext(ctx, `
	UPDATE trials SET total_batches = GREATEST(total_batches, $2)
	WHERE id = $1;
	`, m.TrialId, m.StepsCompleted); err != nil {
			return rollbacks, errors.Wrap(err, "updating trial total batches")
		}
	default: // no rollbacks happened.
		if _, ok := summaryMetrics[metricsJSONPath]; !ok {
			summaryMetrics[metricsJSONPath] = map[string]any{}
		}
		summaryMetrics[metricsJSONPath] = calculateNewSummaryMetrics(
			summaryMetrics[metricsJSONPath].(map[string]any),
			m.Metrics.AvgMetrics,
		)

		var latestValidationID *int
		if isValidation {
			var searcherMetric *string
			if err := tx.QueryRowContext(ctx, `
		SELECT experiments.config->'searcher'->>'metric' AS metric_name
		FROM experiments
		JOIN trials t ON t.experiment_id = experiments.id
		WHERE t.id = $1`, int(m.TrialId)).Scan(&searcherMetric); err != nil {
				return rollbacks, fmt.Errorf("getting trial's searcher metric: %w", err)
			}
			if searcherMetric != nil &&
				m.Metrics.AvgMetrics.Fields[*searcherMetric].AsInterface() != nil {
				latestValidationID = &metricRowID
			}
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE trials SET total_batches = GREATEST(total_batches, $2),
summary_metrics = $3, summary_metrics_timestamp = NOW(),
latest_validation_id = coalesce($4, latest_validation_id)
WHERE id = $1;
`, m.TrialId, m.StepsCompleted, summaryMetrics, latestValidationID); err != nil {
			return rollbacks, errors.Wrap(err, "updating trial total batches")
		}
	}

	if isValidation {
		if err := setTrialBestValidation(
			tx, int(m.TrialId),
			int(m.TrialRunId),
			int(m.StepsCompleted)); err != nil {
			return rollbacks, errors.Wrap(err, "updating trial best validation")
		}
	}
	return rollbacks, nil
}

// addTrialMetrics inserts a set of trial metrics to the database.
func (db *PgDB) addTrialMetrics(
	ctx context.Context, m *trialv1.TrialMetrics, pType MetricPartitionType,
	mType *string,
) (rollbacks int, err error) {
	switch v := m.Metrics.AvgMetrics.Fields["epoch"].AsInterface().(type) {
	case float64, nil:
	default:
		return 0, fmt.Errorf("cannot add metric with non numeric 'epoch' value got %v", v)
	}
	return rollbacks, db.withTransaction(fmt.Sprintf("add trial metrics %s", pType),
		func(tx *sqlx.Tx) error {
			rollbacks, err = db._addTrialMetricsTx(ctx, tx, m, pType, mType)
			return err
		})
}

const (
	// InfPostgresString how we store infinity in JSONB in postgres.
	InfPostgresString = "Infinity"
	// NegInfPostgresString how we store -infinity in JSONB in postgres.
	NegInfPostgresString = "-Infinity"
	// NaNPostgresString how we store NaN in JSONB in postgres.
	NaNPostgresString = "NaN"

	// MetricTypeString is the summary metric type for string or mixed types.
	MetricTypeString = "string"
	// MetricTypeNumber is the summary metric type for floats or ints.
	MetricTypeNumber = "number"
	// MetricTypeBool is the summary metric type for boolean.
	MetricTypeBool = "boolean"
	// MetricTypeDate is the summary metric type for date metrics.
	MetricTypeDate = "date"
	// MetricTypeObject is the summary metric type for object types.
	MetricTypeObject = "object"
	// MetricTypeArray is the summary metric type for array types.
	MetricTypeArray = "array"
	// MetricTypeNull is the summary metric type for array types.
	MetricTypeNull = "null"
)

func jsonAnyToFloat(v any) float64 {
	if s, ok := v.(string); ok {
		if f, isSpecial := stringToSpecialFloats(s); isSpecial {
			return f
		}
	}

	if f, ok := v.(float64); ok {
		return f
	}

	log.Errorf("summary metric value expected as float instead got %T %v", v, v)
	return 0.0
}

func stringToSpecialFloats(s string) (float64, bool) {
	switch s {
	case NaNPostgresString:
		return math.NaN(), true
	case InfPostgresString:
		return math.Inf(1), true
	case NegInfPostgresString:
		return math.Inf(-1), true
	default:
		return 0.0, false
	}
}

func replaceSpecialFloatsWithString(v any) any {
	if f, ok := v.(float64); ok {
		switch {
		case math.IsNaN(f):
			return NaNPostgresString
		case math.IsInf(f, 1.0):
			return InfPostgresString
		case math.IsInf(f, -1.0):
			return NegInfPostgresString
		}
	}
	return v
}

var pythonISOFormatRegex = regexp.MustCompile(
	`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$`)

func calculateNewSummaryMetrics(
	summaryMetrics model.JSONObj, metrics *structpb.Struct,
) model.JSONObj {
	// Calculate numeric metrics.
	for metricName, metric := range metrics.Fields {
		// Get type of provided metric.
		metricFloatValue := 0.0
		metricType := ""
		switch metricValue := metric.AsInterface().(type) {
		case float64:
			metricFloatValue = metricValue
			metricType = MetricTypeNumber
		case string:
			switch f, ok := stringToSpecialFloats(metricValue); {
			case ok:
				metricFloatValue = f
				metricType = MetricTypeNumber
			case pythonISOFormatRegex.MatchString(metricValue):
				metricType = MetricTypeDate
			default:
				metricType = MetricTypeString
			}
		case bool:
			metricType = MetricTypeBool
		case map[string]any:
			metricType = MetricTypeObject
		case []any:
			metricType = MetricTypeArray
		case nil:
			metricType = MetricTypeNull
		default:
			metricType = MetricTypeString
		}

		// If we haven't seen this metric before just add the type we have.
		var ok bool
		var summaryMetric map[string]any
		if summaryMetric, ok = summaryMetrics[metricName].(map[string]any); !ok {
			summaryMetric = map[string]any{"type": metricType}
		} else if summaryMetric["type"] != metricType {
			// If we have seen this before check if we disagree on types and set to string if we do.
			metricType = "string"
			summaryMetric = map[string]any{"type": metricType}
		}
		summaryMetrics[metricName] = summaryMetric

		if metricType != MetricTypeNumber {
			continue
		}

		// Is this the first time seeing a number metric?
		if _, ok = summaryMetric["count"]; !ok {
			summaryMetric["max"] = replaceSpecialFloatsWithString(metricFloatValue)
			summaryMetric["min"] = replaceSpecialFloatsWithString(metricFloatValue)
			summaryMetric["sum"] = replaceSpecialFloatsWithString(metricFloatValue)
			summaryMetric["count"] = 1
		} else {
			summaryMetric["min"] = replaceSpecialFloatsWithString(
				math.Min(jsonAnyToFloat(summaryMetric["min"]), metricFloatValue))
			summaryMetric["max"] = replaceSpecialFloatsWithString(
				math.Max(jsonAnyToFloat(summaryMetric["max"]), metricFloatValue))
			summaryMetric["sum"] = replaceSpecialFloatsWithString(
				jsonAnyToFloat(summaryMetric["sum"]) + metricFloatValue)
			// Go parsing odditity treats JSON whole numbers as floats.
			summaryMetric["count"] = int(jsonAnyToFloat(summaryMetric["count"])) + 1
		}
	}

	// Add last value for all metrics provided.
	for metricName, sumMetric := range summaryMetrics {
		metric, ok := sumMetric.(map[string]any)
		if !ok {
			// Should not happen.
			log.Errorf("summary metric %T %+v is not a map", sumMetric, sumMetric)
			continue
		}

		metric["last"] = replaceSpecialFloatsWithString(metrics.Fields[metricName])
	}

	return summaryMetrics
}

// AddCheckpointMetadata persists metadata for a completed checkpoint to the database.
func AddCheckpointMetadata(ctx context.Context, m *model.CheckpointV2) error {
	var size int64
	for _, v := range m.Resources {
		size += v
	}
	m.Size = size

	err := Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(m).Exec(context.TODO()); err != nil {
			return errors.Wrap(err, "inserting checkpoint")
		}

		if err := UpdateCheckpointSizeTx(ctx, tx, []uuid.UUID{m.UUID}); err != nil {
			return errors.Wrap(err, "updating checkpoint size")
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error adding checkpoint metadata: %w", err)
	}

	return nil
}

func checkTrialRunID(ctx context.Context, tx *sqlx.Tx, trialID, runID int32) error {
	var cRunID int
	switch err := tx.QueryRowxContext(ctx, `
SELECT run_id
FROM trials
WHERE id = $1
`, trialID).Scan(&cRunID); {
	case err != nil:
		return errors.Wrap(err, "querying current run")
	case int(runID) != cRunID:
		return api.AsValidationError("invalid run id, %d (reported) != %d (expected)", runID, cRunID)
	default:
		return nil
	}
}

// ValidationByTotalBatches looks up a validation by trial and total batches,
// returning nil if none exists.
func (db *PgDB) ValidationByTotalBatches(trialID, totalBatches int) (*model.TrialMetrics, error) {
	var validation model.TrialMetrics
	// TODO: update to go through `metrics`.
	if err := db.query(`
SELECT id, trial_id, total_batches, end_time, metrics
FROM validations
WHERE trial_id = $1
AND total_batches = $2`, &validation, trialID, totalBatches); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for validation (%v, %v)",
			trialID, totalBatches)
	}
	return &validation, nil
}

// CheckpointByTotalBatches looks up a checkpoint by trial and total batch,
// returning nil if none exists.
func (db *PgDB) CheckpointByTotalBatches(trialID, totalBatches int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT *
FROM checkpoints_view c
WHERE c.trial_id = $1 AND c.steps_completed = $2`, &checkpoint, trialID, totalBatches,
	); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for checkpoint (%v, %v)",
			trialID, totalBatches)
	}
	return &checkpoint, nil
}

// LatestCheckpointForTrial finds the latest completed checkpoint for a trial, returning nil if
// none exists.
func (db *PgDB) LatestCheckpointForTrial(trialID int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT *
FROM checkpoints_view c
WHERE c.trial_id = $1 AND c.state = 'COMPLETED'
ORDER BY c.steps_completed DESC
LIMIT 1`, &checkpoint, trialID); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for latest trial checkpoint (%v)", trialID)
	}
	return &checkpoint, nil
}

// TrialState returns the current state of the given trial.
func (db *PgDB) TrialState(trialID int) (model.State, error) {
	var state model.State
	err := db.sql.QueryRow(`
SELECT state
FROM trials
WHERE id = $1
`, trialID).Scan(&state)
	return state, err
}

// TrialStatus returns the current status of the given trial, including the end time
// without returning all its hparams and other unneeded details. Called in paths hotter
// than TrialByID allows.
func (db *PgDB) TrialStatus(trialID int) (model.State, *time.Time, error) {
	status := struct {
		State   model.State `db:"state"`
		EndTime *time.Time  `db:"end_time"`
	}{}
	err := db.query(`
SELECT state, end_time
FROM trials
WHERE id = $1
`, &status, trialID)
	return status.State, status.EndTime, err
}

// setTrialBestValidation sets `public.trials.best_validation_id` to the `id` of the row in
// `public.validations` corresponding to the trial's best validation.
func setTrialBestValidation(tx *sqlx.Tx, trialID int, trialRunID int, stepsCompleted int) error {
	_, err := tx.Exec(`
WITH const AS (
    SELECT t.id as trial_id,
           config->'searcher'->>'metric' AS metric_name,
           (SELECT
               CASE WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
			   THEN 1
			   ELSE -1 END) AS sign
    FROM experiments e
    INNER JOIN trials t ON t.experiment_id = e.id
  	WHERE t.id = $1
), best_validation AS (
	SELECT
		v.id AS id,
		const.sign * (v.metrics->'validation_metrics'->>const.metric_name)::float8 AS metric,
		(v.metrics->'validation_metrics'->>const.metric_name)::float8 AS searcher_metric_value
	FROM (
		SELECT * FROM validations where id = (select best_validation_id from trials where id = $1)
		UNION ALL
		SELECT * FROM validations
			where trial_id = $1
			and trial_run_id = $2
			and total_batches = $3
	) v, const
	WHERE v.trial_id = $1
	ORDER BY metric ASC
	LIMIT 1
)
UPDATE trials t
SET best_validation_id = (SELECT bv.id FROM best_validation bv),
searcher_metric_value = (SELECT bv.searcher_metric_value FROM best_validation bv),
searcher_metric_value_signed =
(SELECT bv.searcher_metric_value * const.sign FROM best_validation bv, const)
WHERE t.id = $1;
`, trialID, trialRunID, stepsCompleted)
	return errors.Wrapf(err, "error updating best validation for trial %d", trialID)
}
