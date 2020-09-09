WITH const AS (
  SELECT config->'searcher'->>'metric' AS metric_name,
    (
      SELECT CASE
          WHEN coalesce(
            (
              config->'searcher'->>'smaller_is_better'
            )::boolean,
            true
          ) THEN -1 -- so we can order by DESC to get the highest value
          ELSE 1
        END
    ) AS sign,
    t.id AS trial_id
  FROM experiments e
    INNER JOIN trials t ON t.experiment_id = e.id
  WHERE t.id = 20
),
w_validations AS (
  SELECT v.trial_id,
    v.step_id,
    v.end_time,
    v.state,
    (
      v.metrics->'validation_metrics'->>(const.metric_name)
    )::float8 AS searcher_metric,
    (
      (
        v.metrics->'validation_metrics'->>(const.metric_name)
      )::float8 * const.sign
    ) AS searcher_metric_value
  FROM validations v
    JOIN const ON const.trial_id = v.trial_id
  WHERE v.state = 'COMPLETED' -- add condition to check validation exists
),
best_validation AS (
  SELECT v.*
  FROM w_validations v
  ORDER BY v.searcher_metric_value DESC
  LIMIT 1
), latest_validation AS (
  SELECT v.*
  FROM w_validations v
  ORDER BY end_time DESC
  LIMIT 1
), best_checkpoint AS (
  SELECT c.*
  FROM checkpoints c
    JOIN w_validations v ON c.step_id = v.step_id
  WHERE c.trial_id = 20
  ORDER BY v.searcher_metric_value DESC
  LIMIT 1
)
SELECT bv.searcher_metric AS bv_sm,
  lv.searcher_metric AS lv_sm,
  bc.id AS bc_id
FROM trials t
  INNER JOIN best_validation bv ON bv.trial_id = t.id
  INNER JOIN latest_validation lv ON lv.trial_id = t.id
  INNER JOIN best_checkpoint bc ON bc.trial_id = t.id -- WHERE t.id = 20
LIMIT 1
