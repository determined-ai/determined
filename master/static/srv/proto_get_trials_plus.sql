WITH searcher_info AS (
  SELECT config->'searcher'->>'metric' AS metric_name,
    (
      SELECT CASE
          WHEN coalesce(
            (
              config->'searcher'->>'smaller_is_better'
            )::boolean,
            true
          ) THEN 1
          ELSE -1
        END
    ) AS sign,
    t.id AS trial_id
  FROM experiments e
    INNER JOIN trials t ON t.experiment_id = e.id
  WHERE t.id IN (
      SELECT unnest($1::int [])::int
    )
),
trial_validations AS (
  SELECT v.trial_id,
    v.total_batches,
    v.end_time,
    v.state,
    v.metrics,
    (
      (
        v.metrics->'validation_metrics'->>(searcher_info.metric_name)
      )::float8 * searcher_info.sign
    ) AS signed_searcher_metric
  FROM validations v
    INNER JOIN searcher_info ON v.trial_id = searcher_info.trial_id
  WHERE v.state = 'COMPLETED'
    AND (
      v.metrics->'validation_metrics'->>(searcher_info.metric_name)
    ) IS NOT NULL
),
best_validation AS (
  SELECT v.trial_id,
    v.total_batches,
    v.end_time,
    'STATE_' || v.state AS state,
    v.metrics->'validation_metrics' as metrics,
    v.metrics->'num_inputs' as num_inputs
  FROM (
      SELECT v.*,
        ROW_NUMBER() OVER(
          PARTITION BY v.trial_id
          ORDER BY v.signed_searcher_metric ASC
        ) AS rank
      FROM trial_validations v
    ) v
  JOIN searcher_info ON searcher_info.trial_id = v.trial_id
  WHERE v.rank = 1
),
latest_validation AS (
  SELECT v.trial_id,
    v.total_batches,
    v.end_time,
    'STATE_' || v.state AS state,
    v.metrics->'validation_metrics' as metrics,
    v.metrics->'num_inputs' as num_inputs
  FROM (
      SELECT v.*,
        ROW_NUMBER() OVER(
          PARTITION BY v.trial_id
          ORDER BY v.end_time DESC
        ) AS rank
      FROM trial_validations v
    ) v
  JOIN searcher_info ON searcher_info.trial_id = v.trial_id
  WHERE v.rank = 1
),
best_checkpoint AS (
  SELECT
    c.uuid::text AS uuid,
    c.latest_batch AS total_batches,
    c.trial_id,
    c.report_time AS end_time,
    c.resources,
    'STATE_' || c.state AS state
  FROM (
      SELECT c.*,
        ROW_NUMBER() OVER(
          PARTITION BY v.trial_id
          ORDER BY v.signed_searcher_metric ASC
        ) AS rank
      FROM trial_validations v
      INNER JOIN checkpoints_view c ON (
        c.latest_batch = v.total_batches
        AND c.trial_id = v.trial_id
      )
      WHERE c.state = 'COMPLETED'
    ) c
  WHERE c.rank = 1
)
SELECT
  row_to_json(bv)::jsonb - 'trial_id' AS best_validation,
  row_to_json(lv)::jsonb - 'trial_id' AS latest_validation,
  row_to_json(bc)::jsonb - 'trial_id' AS best_checkpoint,
  t.id AS id,
  t.experiment_id,
  'STATE_' || t.state AS state,
  t.start_time,
  t.end_time,
  t.hparams,
  ckpt.uuid AS warm_start_checkpoint_uuid,
  t.task_id,
  (
    SELECT s.total_batches
    FROM steps s
    WHERE s.trial_id = t.id
      AND s.state = 'COMPLETED'
    ORDER BY s.total_batches DESC
    LIMIT 1
  ) AS total_batches_processed,
   t.runner_state,
  (
    SELECT extract(epoch from sum(coalesce(a.end_time, now()) - a.start_time))
    FROM allocations a
    WHERE a.task_id = t.task_id
  ) AS wall_clock_time
FROM searcher_info
  INNER JOIN trials t ON t.id = searcher_info.trial_id
  LEFT JOIN best_validation bv ON bv.trial_id = searcher_info.trial_id
  LEFT JOIN latest_validation lv ON lv.trial_id = searcher_info.trial_id
  LEFT JOIN best_checkpoint bc ON bc.trial_id = searcher_info.trial_id
  LEFT JOIN checkpoints_view ckpt ON ckpt.id = t.warm_start_checkpoint_id
  -- Return the same ordering of IDs given by $1.
  JOIN (
    SELECT *
    FROM unnest($1::int []) WITH ORDINALITY
  ) AS x (id, ordering) ON t.id = x.id
  ORDER BY x.ordering;
