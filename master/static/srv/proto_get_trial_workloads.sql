WITH validations_vt AS (
  SELECT row_to_json(r1) AS validation
  FROM (
      SELECT 'STATE_' || v.state as state,
        v.start_time,
        v.end_time,
        v.metrics->'num_inputs' as num_inputs,
        v.metrics->'validation_metrics' as metrics
      FROM validations v
      WHERE v.trial_id = 20
    ) AS r1
),
trainings_vt AS (
  SELECT row_to_json(r1) AS training
  FROM (
      SELECT s.start_time,
        s.end_time,
        'STATE_' || s.state as state,
        s.metrics->'avg_metrics' as metrics,
        s.metrics->'num_inputs' as num_inputs
      FROM steps s
      WHERE s.trial_id = 20
    ) AS r1
),
checkpoints_vt AS (
  SELECT row_to_json(r1) AS checkpoint
  FROM (
      SELECT 'STATE_' || c.state as state,
        c.start_time,
        c.end_time,
        c.uuid,
        c.resources
      FROM checkpoints c
      WHERE c.trial_id = 20
    ) AS r1
)
SELECT *
FROM trainings_vt t
  FULL JOIN checkpoints_vt c ON false
  FULL JOIN validations_vt v ON false
