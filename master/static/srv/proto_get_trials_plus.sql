WITH trial_ids_ordering(trial_id, ordering) AS (
  -- of the format "VALUES (trial_id_1, 1), (trial_id_2, 2), ... (trial_id_N, N)"
  -- warning, this query cannot be run with no input.
  VALUES %s
),
searcher_info AS (
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
    (config->>'max_restarts')::int AS max_restarts,
    t.id AS trial_id,
    t_ordering.ordering AS ordering
  FROM experiments e
  JOIN trials t ON t.experiment_id = e.id
  JOIN trial_ids_ordering t_ordering ON t.id = t_ordering.trial_id
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
    json_build_object('avg_metrics', v.metrics->'validation_metrics') as metrics,
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
    json_build_object('avg_metrics', v.metrics->'validation_metrics') as metrics,
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
    c.steps_completed AS total_batches,
    c.trial_id,
    c.report_time AS end_time,
    c.resources,
    'STATE_' || c.state AS state
  FROM (
    -- Using `public.checkpoints_view` directly results in performance regressions
    -- identical to those described below.
    SELECT c.*,
      ROW_NUMBER() OVER(
        PARTITION BY c.trial_id
        ORDER BY c.signed_searcher_metric ASC
      ) AS rank
    FROM (
      SELECT old_c.*
      FROM (
          SELECT c.*, v.signed_searcher_metric,
            ROW_NUMBER() OVER(
              PARTITION BY v.trial_id
              ORDER BY v.signed_searcher_metric ASC
            ) AS rank
          FROM trial_validations v
          INNER JOIN checkpoints_old_view c ON (
            c.steps_completed = v.total_batches
            AND c.trial_id = v.trial_id
          )
          WHERE c.state = 'COMPLETED'
        ) old_c
      WHERE old_c.rank = 1
      UNION ALL
      SELECT new_c.*
      FROM (
          SELECT c.*, v.signed_searcher_metric,
            ROW_NUMBER() OVER(
              PARTITION BY v.trial_id
              ORDER BY v.signed_searcher_metric ASC
            ) AS rank
          FROM trial_validations v
          INNER JOIN checkpoints_new_view c ON (
            c.steps_completed = v.total_batches
            AND c.trial_id = v.trial_id
          )
          WHERE c.state = 'COMPLETED'
        ) new_c
      WHERE new_c.rank = 1
    ) c
    WHERE c.rank = 1
  ) c
),
latest_training AS (
  SELECT s.trial_id,
    s.total_batches,
    s.end_time,
    'STATE_' || s.state AS state,
    json_build_object('avg_metrics', s.metrics->'avg_metrics') as metrics
  FROM (
      SELECT s.*,
        ROW_NUMBER() OVER(
          PARTITION BY s.trial_id
          ORDER BY s.end_time DESC
        ) AS rank
      FROM steps s
      INNER JOIN searcher_info ON s.trial_id = searcher_info.trial_id
      WHERE s.state = 'COMPLETED'
    ) s
  JOIN searcher_info ON searcher_info.trial_id = s.trial_id
  WHERE s.rank = 1
)
SELECT
  row_to_json(bv)::jsonb - 'trial_id' AS best_validation,
  row_to_json(lv)::jsonb - 'trial_id' AS latest_validation,
  row_to_json(bc)::jsonb - 'trial_id' AS best_checkpoint,
  row_to_json(lt)::jsonb - 'trial_id' AS latest_training,
  t.id AS id,
  t.experiment_id,
  'STATE_' || t.state AS state,
  t.start_time,
  t.end_time,
  t.hparams,
  coalesce(new_ckpt.uuid, old_ckpt.uuid) AS warm_start_checkpoint_uuid,
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
  ) AS wall_clock_time,
  (
    SELECT sum((jsonb_each).value::text::bigint)
    FROM (
        SELECT jsonb_each(resources) FROM checkpoints_old_view c WHERE c.trial_id = t.id
        UNION ALL
        SELECT jsonb_each(resources) FROM checkpoints_new_view c WHERE c.trial_id = t.id
    ) r
  ) AS total_checkpoint_size,
  -- `restart` count is incremented before `restart <= max_restarts` stop restart check,
  -- so trials in terminal state have restarts = max + 1
  LEAST(t.restarts, max_restarts) as restarts
FROM searcher_info
  INNER JOIN trials t ON t.id = searcher_info.trial_id
  LEFT JOIN best_validation bv ON bv.trial_id = searcher_info.trial_id
  LEFT JOIN latest_validation lv ON lv.trial_id = searcher_info.trial_id
  LEFT JOIN best_checkpoint bc ON bc.trial_id = searcher_info.trial_id
  -- Using `public.checkpoints_view` directly here results in the query planner being unable to push
  -- filters into the union all, resulting in costly scans of steps, validations and checkpoints.
  -- additionally, it joins a lot of stuff we don't need, so just fallback to the actual tables.
  LEFT JOIN raw_checkpoints old_ckpt ON old_ckpt.id = t.warm_start_checkpoint_id
  LEFT JOIN checkpoints_v2 new_ckpt ON new_ckpt.id = t.warm_start_checkpoint_id
  LEFT JOIN latest_training lt ON lt.trial_id = searcher_info.trial_id
  ORDER BY searcher_info.ordering
