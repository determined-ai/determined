CREATE OR REPLACE FUNCTION safe_sum_accumulate(float8, float8, OUT float8)
  RETURNS float8 AS $$
  BEGIN
    -- Check for potential overflow
    BEGIN
      IF $1 IS NULL THEN
        $3 := $2;
      ELSIF $2 IS NULL THEN
        $3 := $1;
      ELSE
        $3 := $1 + $2;
      END IF;
    EXCEPTION
      WHEN numeric_value_out_of_range THEN
        IF $1 < 0 THEN
          $3 := '-Infinity';
        ELSE
          $3 := 'Infinity';
        END IF;
    END;
  END;
$$ LANGUAGE plpgsql;

DROP AGGREGATE IF EXISTS safe_sum(float8);

CREATE AGGREGATE safe_sum(float8) (
  SFUNC = safe_sum_accumulate,
  STYPE = float8
);

-- We need to declare start_timestamp before this migration starts
-- to avoid missing metrics reported while this migration runs in background cases.
DO $$ DECLARE start_timestamp timestamptz := NOW(); BEGIN

-- Safe sum to avoid overflows.
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

-- Validations.
CREATE TEMPORARY TABLE val_metric_values (
  id SERIAL,
  trial_id INT,
  name TEXT,
  value TEXT,
  type TEXT,
  end_time timestamptz
);

CREATE TEMPORARY TABLE val_numeric_aggs (
  id SERIAL,
  trial_id INT,
  name TEXT,
  count INT,
  sum FLOAT8,
  min FLOAT8,
  max FLOAT8
);

CREATE TEMPORARY TABLE val_metric_types (
  id SERIAL,
  trial_id INT,
  name TEXT,
  type TEXT
);

CREATE INDEX val_metric_types_idx ON val_metric_types (trial_id, name);

CREATE TEMPORARY TABLE val_metric_latest (
  id SERIAL,
  trial_id INT,
  name TEXT,
  value jsonb
);

CREATE INDEX metric_latest_idx ON val_metric_latest (trial_id, name);

CREATE TEMPORARY TABLE val_summary_metrics (
  id SERIAL,
  trial_id INT,
  summary_metrics JSONB
);

-- Extract training metrics.
INSERT INTO val_metric_values(trial_id, name, value, type, end_time)
SELECT
    trial_id AS trial_id,
    key AS name,
    CASE value
        WHEN '"NaN"' THEN 'NaN'
        WHEN '"Infinity"' THEN 'Infinity'
        WHEN '"-Infinity"' THEN '-Infinity'
        ELSE value::text
    END AS value,
    CASE
        WHEN jsonb_typeof(value) = 'string' THEN
            CASE
                WHEN value::text = '"Infinity"'::text THEN 'number'
                WHEN value::text = '"-Infinity"'::text THEN 'number'
                WHEN value::text = '"NaN"'::text THEN 'number'
                WHEN value::text ~
                    '^"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?"$' THEN 'date'
                ELSE 'string'
            END
        ELSE jsonb_typeof(value)::text
    END AS type,
    end_time AS end_time
FROM (
    SELECT
        validations.trial_id,
        (jsonb_each(metrics->'validation_metrics')).key,
        (jsonb_each(metrics->'validation_metrics')).value,
        validations.end_time
    FROM validations
    JOIN trials ON trials.id = validations.trial_id
    WHERE trials.summary_metrics_timestamp IS NULL
) AS subquery;

-- Numeric aggregates.
INSERT INTO val_numeric_aggs(trial_id, name, count, sum, min, max)
SELECT
    trial_id AS trial_id,
    name AS name,
    COUNT(*) AS count,
    safe_sum(value::double precision) AS sum,
    MIN(value::double precision) AS min,
    MAX(value::double precision) AS max
FROM val_metric_values
WHERE type = 'number'
GROUP BY trial_id, name;

-- Types.
INSERT INTO val_metric_types(trial_id, name, type)
SELECT
    trial_id AS trial_id,
    name AS name,
    CASE
        WHEN COUNT(DISTINCT type) = 1 THEN MAX(type)
        ELSE 'string'
    END AS type
FROM val_metric_values
GROUP BY trial_id, name;

-- Latest.
INSERT INTO val_metric_latest(trial_id, name, value)
SELECT
    s.trial_id AS trial_id,
    unpacked.key as name,
    unpacked.value as value
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
WHERE s.rank = 1;

-- Summary metrics.
INSERT INTO val_summary_metrics(trial_id, summary_metrics)
SELECT
    trial_id, jsonb_collect(jsonb_build_object(
        name, jsonb_build_object(
        'count', CASE WHEN sub.type = 'number' THEN sub.count ELSE 0 END,
        'sum', CASE WHEN sub.type = 'number' THEN sub.sum ELSE 0 END,
        'min', CASE WHEN sub.type = 'number' THEN
            CASE WHEN sub.max = 'NaN'::double precision
                THEN 'NaN'::double precision ELSE sub.min END
            ELSE 0 END,
        'max', CASE WHEN sub.type = 'number' THEN sub.max ELSE 0 END,
        'last', sub.latest,
        'type', sub.type
    )
)) as summary_metrics
FROM (SELECT
    val_metric_types.trial_id,
    val_metric_types.name,
    count,
    sum,
    min,
    max,
    val_metric_types.type AS type,
    val_metric_latest.value AS latest
FROM val_metric_types
LEFT JOIN val_numeric_aggs ON
     val_numeric_aggs.trial_id = val_metric_types.trial_id AND
     val_numeric_aggs.name = val_metric_types.name
LEFT JOIN val_metric_latest ON
     val_metric_types.trial_id = val_metric_latest.trial_id AND
     val_metric_types.name = val_metric_latest.name) sub
GROUP BY trial_id;

-- Training.
CREATE TEMPORARY TABLE train_metric_values (
  id SERIAL,
  trial_id INT,
  name TEXT,
  value TEXT,
  type TEXT,
  end_time timestamptz
);

CREATE TEMPORARY TABLE train_numeric_aggs (
  id SERIAL,
  trial_id INT,
  name TEXT,
  count INT,
  sum FLOAT8,
  min FLOAT8,
  max FLOAT8
);

CREATE TEMPORARY TABLE train_metric_types (
  id SERIAL,
  trial_id INT,
  name TEXT,
  type TEXT
);

CREATE INDEX train_metric_types_idx ON train_metric_types (trial_id, name);

CREATE TEMPORARY TABLE train_metric_latest (
  id SERIAL,
  trial_id INT,
  name TEXT,
  value jsonb
);

CREATE INDEX train_metric_latest_idx ON train_metric_latest (trial_id, name);

CREATE TEMPORARY TABLE train_summary_metrics (
  id SERIAL,
  trial_id INT,
  summary_metrics JSONB
);

-- Extract training metrics.
INSERT INTO train_metric_values(trial_id, name, value, type, end_time)
SELECT
    trial_id AS trial_id,
    key AS name,
    CASE value
        WHEN '"NaN"' THEN 'NaN'
        WHEN '"Infinity"' THEN 'Infinity'
        WHEN '"-Infinity"' THEN '-Infinity'
        ELSE value::text
    END AS value,
    CASE
        WHEN jsonb_typeof(value) = 'string' THEN
            CASE
                WHEN value::text = '"Infinity"'::text THEN 'number'
                WHEN value::text = '"-Infinity"'::text THEN 'number'
                WHEN value::text = '"NaN"'::text THEN 'number'
                WHEN value::text ~
                    '^"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?"$' THEN 'date'
                ELSE 'string'
            END
        ELSE jsonb_typeof(value)::text
    END AS type,
    end_time AS end_time
FROM (
    SELECT
        steps.trial_id,
        (jsonb_each(metrics->'avg_metrics')).key,
        (jsonb_each(metrics->'avg_metrics')).value,
        steps.end_time
    FROM steps
    JOIN trials ON trials.id = steps.trial_id
    WHERE trials.summary_metrics_timestamp IS NULL
) AS subquery;

-- Numeric aggregates.
INSERT INTO train_numeric_aggs(trial_id, name, count, sum, min, max)
SELECT
    trial_id AS trial_id,
    name AS name,
    COUNT(*) AS count,
    safe_sum(value::double precision) AS sum,
    MIN(value::double precision) AS min,
    MAX(value::double precision) AS max
FROM train_metric_values
WHERE type = 'number'
GROUP BY trial_id, name;

-- Types.
INSERT INTO train_metric_types(trial_id, name, type)
SELECT
    trial_id AS trial_id,
    name AS name,
    CASE
        WHEN COUNT(DISTINCT type) = 1 THEN MAX(type)
        ELSE 'string'
    END AS type
FROM train_metric_values
GROUP BY trial_id, name;

-- Latest.
INSERT INTO train_metric_latest(trial_id, name, value)
SELECT
    s.trial_id AS trial_id,
    unpacked.key as name,
    unpacked.value as value
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
WHERE s.rank = 1;

-- Summary metrics.
INSERT INTO train_summary_metrics(trial_id, summary_metrics)
SELECT
    trial_id, jsonb_collect(jsonb_build_object(
        name, jsonb_build_object(
        'count', CASE WHEN sub.type = 'number' THEN sub.count ELSE 0 END,
        'sum', CASE WHEN sub.type = 'number' THEN sub.sum ELSE 0 END,
        'min', CASE WHEN sub.type = 'number' THEN
            CASE WHEN sub.max = 'NaN'::double precision
                THEN 'NaN'::double precision ELSE sub.min END
            ELSE 0 END,
        'max', CASE WHEN sub.type = 'number' THEN sub.max ELSE 0 END,
        'last', sub.latest,
        'type', sub.type
    )
)) as summary_metrics
FROM (SELECT
    train_metric_types.trial_id,
    train_metric_types.name,
    count,
    sum,
    min,
    max,
    train_metric_types.type AS type,
    train_metric_latest.value AS latest
FROM train_metric_types
LEFT JOIN train_numeric_aggs ON
     train_numeric_aggs.trial_id = train_metric_types.trial_id AND
     train_numeric_aggs.name = train_metric_types.name
LEFT JOIN train_metric_latest ON
     train_metric_types.trial_id = train_metric_latest.trial_id AND
     train_metric_types.name = train_metric_latest.name) sub
GROUP BY trial_id;

UPDATE trials SET
    summary_metrics = (CASE
        WHEN tsm.summary_metrics IS NOT NULL AND vsm.summary_metrics IS NOT NULL THEN
            jsonb_build_object(
                'avg_metrics', tsm.summary_metrics,
                'validation_metrics', vsm.summary_metrics
            )
        WHEN tsm.summary_metrics IS NOT NULL THEN
            jsonb_build_object(
                'avg_metrics', tsm.summary_metrics
            )
        WHEN vsm.summary_metrics IS NOT NULL THEN jsonb_build_object(
                'validation_metrics', vsm.summary_metrics
           )
        ELSE '{}'::jsonb END)
FROM train_summary_metrics tsm
FULL OUTER JOIN val_summary_metrics vsm ON tsm.trial_id = vsm.trial_id
WHERE coalesce(tsm.trial_id, vsm.trial_id) = trials.id;

DROP TABLE train_metric_values;
DROP TABLE train_numeric_aggs;
DROP TABLE train_metric_types;
DROP TABLE train_metric_latest;
DROP TABLE train_summary_metrics;

DROP TABLE val_metric_values;
DROP TABLE val_numeric_aggs;
DROP TABLE val_metric_types;
DROP TABLE val_metric_latest;
DROP TABLE val_summary_metrics;

-- Set the timestamp to the time we started this migration.
UPDATE trials SET
    summary_metrics_timestamp = start_timestamp;

END$$;
