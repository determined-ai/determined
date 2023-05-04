-- We need to declare start_timestamp before this migration starts
-- to avoid missing metrics reported while this migration runs in background cases.
DO $$ DECLARE start_timestamp timestamptz := NOW(); BEGIN

ALTER TABLE trials
    ADD COLUMN IF NOT EXISTS summary_metrics jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS summary_metrics_timestamp timestamptz;

-- Invalidate summary_metrics_timestamp for trials that have a metric added since.
WITH max_training as (
     SELECT trial_id, max(steps.end_time) as last_reported_metric FROM steps
     JOIN trials ON trials.id = trial_id
     WHERE summary_metrics_timestamp IS NOT NULL
     GROUP BY trial_id
)
UPDATE trials SET summary_metrics_timestamp = NULL FROM max_training WHERE
    max_training.trial_id = trials.id AND
    summary_metrics_timestamp IS NOT NULL AND
    last_reported_metric > summary_metrics_timestamp;

WITH max_validation as (
     SELECT trial_id, max(validations.end_time) as last_reported_metric FROM validations
     JOIN trials ON trials.id = trial_id
     WHERE summary_metrics_timestamp IS NOT NULL
     GROUP BY trial_id
)
UPDATE trials SET summary_metrics_timestamp = NULL FROM max_validation WHERE
     max_validation.trial_id = trials.id AND
     summary_metrics_timestamp IS NOT NULL AND
     last_reported_metric > summary_metrics_timestamp;

-- Returns pairs of metric names and trial_ids and their datatypes.
WITH training_trial_metrics as (
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
        WHEN jsonb_typeof(metrics->'avg_metrics'->name) = 'string' THEN
            CASE
                WHEN (metrics->'avg_metrics'->name)::text = '"Infinity"'::text THEN 'number'
                WHEN (metrics->'avg_metrics'->name)::text = '"-Infinity"'::text THEN 'number'
                WHEN (metrics->'avg_metrics'->name)::text = '"NaN"'::text THEN 'number'
                WHEN metrics->'avg_metrics'->>name ~
                    '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$' THEN 'date'
                ELSE 'string'
            END
        ELSE jsonb_typeof(metrics->'avg_metrics'->name)
    END as metric_type,
    trial_id,
    count(1) as entries
    FROM (
        SELECT DISTINCT
        jsonb_object_keys(s.metrics->'avg_metrics') as name
        FROM steps s
        JOIN trials ON s.trial_id = trials.id
        WHERE trials.summary_metrics_timestamp IS NULL
    ) names, steps
    JOIN trials ON trial_id = trials.id
    WHERE trials.summary_metrics_timestamp IS NULL
    GROUP BY name, metric_type, trial_id
) typed
where metric_type IS NOT NULL
GROUP BY name, trial_id
ORDER BY trial_id, name
),
-- Filters to only numeric metrics.
training_numeric_trial_metrics as (
SELECT name, trial_id
FROM training_trial_metrics
WHERE metric_type = 'number'
),
-- Calculates count, sum, min, max on each numeric metric name and trial ID pair.
-- Also adds just the name for non numeric metrics to ensure we record every metric.
training_trial_metric_aggs as (
SELECT
    name,
    ntm.trial_id,
    count(1) as count_agg,
    sum((steps.metrics->'avg_metrics'->>name)::double precision) as sum_agg,
    min((steps.metrics->'avg_metrics'->>name)::double precision) as min_agg,
    max((steps.metrics->'avg_metrics'->>name)::double precision) as max_agg,
    'number' as metric_type
FROM training_numeric_trial_metrics ntm INNER JOIN steps
ON steps.trial_id=ntm.trial_id
WHERE steps.metrics->'avg_metrics'->name IS NOT NULL
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
FROM training_trial_metrics
WHERE metric_type != 'number'
),
-- Gets the last reported metric for each trial. Note if we report
-- {"a": 1} and {"b": 1} we consider {"b": 1} to be the last reported
-- metric and "a"'s last will be NULL.
latest_training as (
  SELECT s.trial_id,
    unpacked.key as name,
    unpacked.value as latest_value
  FROM (
      SELECT s.*,
        ROW_NUMBER() OVER(
          PARTITION BY s.trial_id
          ORDER BY s.end_time DESC
        ) as rank
      FROM steps s
      JOIN trials ON s.trial_id = trials.id
      WHERE trials.summary_metrics_timestamp IS NULL
    ) s, jsonb_each(s.metrics->'avg_metrics') unpacked
  WHERE s.rank = 1
),
-- Adds the last reported metric to training the aggregation.
training_combined_latest_agg as (SELECT
    coalesce(lt.trial_id, tma.trial_id) as trial_id,
    coalesce(lt.name, tma.name) as name,
    tma.count_agg,
    tma.sum_agg,
    tma.min_agg,
    tma.max_agg,
    lt.latest_value,
    tma.metric_type
FROM latest_training lt FULL OUTER JOIN training_trial_metric_aggs tma ON
    lt.trial_id = tma.trial_id AND lt.name = tma.name
),
-- Turns each rows into a JSONB object.
training_trial_metrics_final as (
    SELECT
        trial_id, jsonb_collect(jsonb_build_object(
            name, jsonb_build_object(
                'count', count_agg,
                'sum', sum_agg,
                'min', CASE WHEN max_agg = 'NaN'::double precision THEN 'NaN'::double precision ELSE min_agg END,
                'max', max_agg,
                'last', latest_value,
                'type', metric_type
            )
        )) as training_metrics
    FROM training_combined_latest_agg
    GROUP BY trial_id
),
-- We repeat the same process as above to validation metrics.
validation_trial_metrics as (
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
        WHEN jsonb_typeof(metrics->'validation_metrics'->name) = 'string' THEN
            CASE
                WHEN (metrics->'validation_metrics'->name)::text = '"Infinity"'::text THEN 'number'
                WHEN (metrics->'validation_metrics'->name)::text = '"-Infinity"'::text THEN 'number'
                WHEN (metrics->'validation_metrics'->name)::text = '"NaN"'::text THEN 'number'
                WHEN metrics->'validation_metrics'->>name ~
                    '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$' THEN 'date'
                ELSE 'string'
            END
        ELSE jsonb_typeof(metrics->'validation_metrics'->name)
    END as metric_type,
    trial_id,
    count(1) as entries
    FROM (
        SELECT DISTINCT
        jsonb_object_keys(s.metrics->'validation_metrics') as name
        FROM validations s
        JOIN trials ON s.trial_id = trials.id
        WHERE trials.summary_metrics_timestamp IS NULL
    ) names, validations
    JOIN trials ON trial_id = trials.id
    WHERE trials.summary_metrics_timestamp IS NULL
    GROUP BY name, metric_type, trial_id
) typed
where metric_type is not NULL
GROUP BY name, trial_id
ORDER BY trial_id, name
),
validation_numeric_trial_metrics as (
SELECT name, trial_id
FROM validation_trial_metrics
WHERE metric_type = 'number'
),
validation_trial_metric_aggs as (
SELECT
    name,
    ntm.trial_id,
    count(1) as count_agg,
    sum((validations.metrics->'validation_metrics'->>name)::double precision) as sum_agg,
    min((validations.metrics->'validation_metrics'->>name)::double precision) as min_agg,
    max((validations.metrics->'validation_metrics'->>name)::double precision) as max_agg,
    'number' as metric_type
FROM validation_numeric_trial_metrics ntm INNER JOIN validations
ON validations.trial_id=ntm.trial_id
WHERE validations.metrics->'validation_metrics'->name IS NOT NULL
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
FROM validation_trial_metrics
WHERE metric_type != 'number'
),
latest_validation as (
    SELECT s.trial_id,
        unpacked.key as name,
        unpacked.value as latest_value
    FROM (
        SELECT s.*,
            ROW_NUMBER() OVER(
                PARTITION BY s.trial_id
                ORDER BY s.end_time DESC
            ) as rank
        FROM validations s
        JOIN trials ON s.trial_id = trials.id
        WHERE trials.summary_metrics_timestamp IS NULL
    ) s, jsonb_each(s.metrics->'validation_metrics') unpacked
    WHERE s.rank = 1
),
validation_combined_latest_agg as (SELECT
    coalesce(lt.trial_id, tma.trial_id) as trial_id,
    coalesce(lt.name, tma.name) as name,
    tma.count_agg,
    tma.sum_agg,
    tma.min_agg,
    tma.max_agg,
    tma.metric_type,
    lt.latest_value
FROM latest_validation lt FULL OUTER JOIN validation_trial_metric_aggs tma ON
    lt.trial_id = tma.trial_id AND lt.name = tma.name
),
validation_trial_metrics_final as (
    SELECT
        trial_id, jsonb_collect(jsonb_build_object(
            name, jsonb_build_object(
                'count', count_agg,
                'sum', sum_agg,
                'min', CASE WHEN max_agg = 'NaN'::double precision THEN 'NaN'::double precision ELSE min_agg END,
                'max', max_agg,
                'last', latest_value,
                'type', metric_type
            )
        )) as validation_metrics
    FROM validation_combined_latest_agg
    GROUP BY trial_id
),
-- Combine both training and validation metrics into a single JSON object.
validation_training_combined_json as (
    SELECT
    coalesce(ttm.trial_id, vtm.trial_id) as trial_id,
    (CASE
        WHEN ttm.training_metrics IS NOT NULL AND vtm.validation_metrics IS NOT NULL THEN
            jsonb_build_object(
                'avg_metrics', ttm.training_metrics,
                'validation_metrics', vtm.validation_metrics
            )
        WHEN ttm.training_metrics IS NOT NULL THEN
            jsonb_build_object(
                'avg_metrics', ttm.training_metrics
            )
        WHEN vtm.validation_metrics IS NOT NULL THEN jsonb_build_object(
                'validation_metrics', vtm.validation_metrics
           )
        ELSE '{}'::jsonb END) as summary_metrics
    FROM training_trial_metrics_final ttm FULL OUTER JOIN validation_trial_metrics_final vtm
    ON ttm.trial_id = vtm.trial_id
)
-- Updates trials with this training and validation object.
UPDATE trials SET
    summary_metrics = vtcj.summary_metrics
FROM validation_training_combined_json vtcj WHERE vtcj.trial_id = trials.id;

-- Set the timestamp to the time we started this migration.
UPDATE trials SET
    summary_metrics_timestamp = start_timestamp;

END$$;
