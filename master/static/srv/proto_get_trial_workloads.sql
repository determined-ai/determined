WITH validations_vt AS (
  SELECT row_to_json(r1) AS validation, total_batches
  FROM (
      SELECT 'STATE_' || v.state as state,
        v.end_time,
        v.total_batches,
        v.metrics->'num_inputs' as num_inputs,
        v.metrics->'validation_metrics' as metrics
      FROM validations v
      WHERE v.trial_id = $1
    ) AS r1
),
trainings_vt AS (
  SELECT row_to_json(r1) AS training, total_batches
  FROM (
      SELECT s.end_time,
        'STATE_' || s.state as state,
        s.metrics->'avg_metrics' as metrics,
        s.metrics->'num_inputs' as num_inputs,
        s.total_batches
      FROM steps s
      WHERE s.trial_id = $1
    ) AS r1
),
checkpoints_vt AS (
  SELECT row_to_json(r1) AS checkpoint, total_batches
  FROM (
      SELECT
        'STATE_' || c.state AS state,
        c.report_time as end_time,
        c.uuid,
        c.latest_batch as total_batches,
        c.resources
      FROM checkpoints_view c
      WHERE c.trial_id = $1
    ) AS r1
)
SELECT v.validation::jsonb AS validation,
  t.training::jsonb AS training,
  c.checkpoint::jsonb AS checkpoint
FROM trainings_vt t
  FULL JOIN checkpoints_vt c ON false
  FULL JOIN validations_vt v ON false
ORDER BY coalesce(
    t.total_batches,
    v.total_batches,
    c.total_batches
  ) ASC
