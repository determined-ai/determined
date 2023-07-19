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
    v.metrics,
    (
      (
        v.metrics->'validation_metrics'->>(searcher_info.metric_name)
      )::float8 * searcher_info.sign
    ) AS signed_searcher_metric
  FROM validations v
    INNER JOIN searcher_info ON v.trial_id = searcher_info.trial_id
  WHERE (
    v.metrics->'validation_metrics'->>(searcher_info.metric_name)
  ) IS NOT NULL
),
best_validation AS (
  SELECT v.trial_id,
    v.total_batches,
    v.end_time,
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
    json_build_object('avg_metrics', v.metrics->'validation_metrics') as metrics,
    v.metrics->'num_inputs' as num_inputs
  FROM validations v
  JOIN searcher_info ON searcher_info.trial_id = v.trial_id
  JOIN trials t ON t.id = v.trial_id AND t.latest_validation_id = v.id
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
      SELECT new_c.*
      FROM (
          SELECT c.*, v.signed_searcher_metric,
            ROW_NUMBER() OVER(
              PARTITION BY v.trial_id
              ORDER BY v.signed_searcher_metric ASC
            ) AS rank
          FROM trial_validations v
          INNER JOIN checkpoints_view c ON (
            c.steps_completed = v.total_batches
            AND c.trial_id = v.trial_id
          )
          WHERE c.state = 'COMPLETED'
        ) new_c
      WHERE new_c.rank = 1
    ) c
    WHERE c.rank = 1
  ) c
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
  new_ckpt.uuid AS warm_start_checkpoint_uuid,
  t.task_id,
  t.checkpoint_size AS total_checkpoint_size,
  t.checkpoint_count,
  t.total_batches AS total_batches_processed,
   t.runner_state,
   t.summary_metrics AS summary_metrics, 
  (
    SELECT extract(epoch from sum(coalesce(a.end_time, now()) - a.start_time))
    FROM allocations a
    WHERE a.task_id = t.task_id
  ) AS wall_clock_time,
  -- `restart` count is incremented before `restart <= max_restarts` stop restart check,
  -- so trials in terminal state have restarts = max + 1
  LEAST(t.restarts, max_restarts) as restarts
FROM searcher_info
  INNER JOIN trials t ON t.id = searcher_info.trial_id
  LEFT JOIN best_validation bv ON bv.trial_id = searcher_info.trial_id
  LEFT JOIN latest_validation lv ON lv.trial_id = searcher_info.trial_id
  LEFT JOIN best_checkpoint bc ON bc.trial_id = searcher_info.trial_id
  LEFT JOIN checkpoints_v2 new_ckpt ON new_ckpt.id = t.warm_start_checkpoint_id
  ORDER BY searcher_info.ordering
