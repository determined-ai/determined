WITH validations_vt AS (
  SELECT row_to_json(r1) AS validation, total_batches
  FROM (
      SELECT 'STATE_' || v.state as state,
        v.start_time,
        v.end_time,
        s.num_batches,
        s.prior_batches_processed,
        v.total_batches,
        v.metrics->'num_inputs' as num_inputs,
        v.metrics->'validation_metrics' as metrics
      FROM validations v
        INNER JOIN steps s ON v.trial_id = s.trial_id
        AND v.total_batches = s.total_batches
      WHERE v.trial_id = $1
    ) AS r1
),
trainings_vt AS (
  SELECT row_to_json(r1) AS training, total_batches
  FROM (
      SELECT s.start_time,
        s.end_time,
        'STATE_' || s.state as state,
        s.num_batches,
        s.prior_batches_processed,
        s.total_batches,
        s.metrics->'avg_metrics' as metrics,
        s.metrics->'num_inputs' as num_inputs
      FROM steps s
      WHERE s.trial_id = $1
    ) AS r1
),
checkpoints_vt AS (
  SELECT row_to_json(r1) AS checkpoint, total_batches
  FROM (
      SELECT 'STATE_' || c.state as state,
        c.start_time,
        c.end_time,
        c.uuid,
        c.total_batches,
        s.num_batches,
        s.prior_batches_processed,
        c.resources
      FROM checkpoints c
        INNER JOIN steps s ON c.trial_id = s.trial_id
        AND c.total_batches = s.total_batches
      WHERE c.trial_id = $1
    ) AS r1
)
SELECT v.validation::jsonb - 'total_batches' AS validation,
  t.training::jsonb - 'total_batches' AS training,
  c.checkpoint::jsonb - 'total_batches' AS checkpoint
FROM trainings_vt t
  FULL JOIN checkpoints_vt c ON false
  FULL JOIN validations_vt v ON false
ORDER BY coalesce(
    t.total_batches,
    v.total_batches,
    c.total_batches
  ) ASC
