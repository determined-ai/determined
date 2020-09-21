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
  WHERE t.id IN (
      SELECT unnest(string_to_array($1, ','))::int
    )
),
w_validations AS (
  SELECT v.trial_id,
    v.step_id,
    v.start_time,
    v.end_time,
    v.state,
    (
      (
        v.metrics->'validation_metrics'->>(const.metric_name)
      )::float8 * const.sign
    ) AS signed_searcher_metric
  FROM validations v
    INNER JOIN const ON v.trial_id = const.trial_id
  WHERE v.state = 'COMPLETED'
    AND (
      v.metrics->'validation_metrics'->>(const.metric_name)
    ) IS NOT NULL
),
best_validation AS (
  SELECT s.trial_Id,
    s.step_id,
    s.start_time,
    s.end_time,
    'STATE_' || s.state AS state,
    s.signed_searcher_metric as searcher_metric -- FIXME
  FROM (
      SELECT v.*,
        ROW_NUMBER() OVER(
          PARTITION BY v.trial_id
          ORDER BY v.signed_searcher_metric DESC
        ) AS rk
      FROM w_validations v
    ) s
  WHERE s.rk = 1
),
latest_validation AS (
  SELECT s.trial_Id,
    s.step_id,
    s.start_time,
    s.end_time,
    'STATE_' || s.state AS state,
    s.signed_searcher_metric as searcher_metric -- FIXME
  FROM (
      SELECT v.*,
        ROW_NUMBER() OVER(
          PARTITION BY v.trial_id
          ORDER BY v.end_time DESC
        ) AS rk
      FROM w_validations v
    ) s
  WHERE s.rk = 1
),
best_checkpoint AS (
  SELECT s.uuid::text AS uuid,
    s.trial_id,
    s.step_id,
    s.start_time AS start_time,
    s.end_time AS end_time,
    s.resources AS resources,
    'STATE_' || s.state AS state
  FROM (
      SELECT c.*,
        ROW_NUMBER() OVER(
          PARTITION BY v.trial_id
          ORDER BY v.signed_searcher_metric DESC
        ) AS rk
      FROM w_validations v
        INNER JOIN checkpoints c ON (
          c.step_id = v.step_id
          AND c.trial_id = v.trial_id
        )
      WHERE c.state = 'COMPLETED'
    ) s
  WHERE s.rk = 1
)
SELECT row_to_json(bv) AS best_validation,
  row_to_json(lv) AS latest_validation,
  row_to_json(bc) AS best_checkpoint,
  t.id AS id,
  t.experiment_id,
  'STATE_' || t.state AS state,
  t.start_time,
  t.end_time,
  t.hparams,
  (
    SELECT s.prior_batches_processed + s.num_batches
    FROM steps s
    WHERE s.trial_id = t.id
      AND s.state = 'COMPLETED'
    ORDER BY s.id DESC
    LIMIT 1
  ) AS total_batches_processed
FROM const
  INNER JOIN trials t ON t.id = const.trial_id
  LEFT JOIN best_validation bv ON bv.trial_id = const.trial_id
  LEFT JOIN latest_validation lv ON lv.trial_id = const.trial_id
  LEFT JOIN best_checkpoint bc ON bc.trial_id = const.trial_id
