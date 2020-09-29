SELECT s.id,
  s.num_batches,
  s.prior_batches_processed,
  (
    SELECT row_to_json(r1)
    FROM (
        SELECT 'STATE_' || v.state as state,
          v.start_time,
          v.end_time,
          v.metrics->'num_inputs' as num_inputs,
          v.metrics->'validation_metrics' as metrics
        FROM validations v
        WHERE v.trial_id = s.trial_id
          AND v.step_id = s.id
      ) r1
  ) AS validation,
  (
    SELECT row_to_json(r2)
    FROM (
        SELECT 'STATE_' || c.state as state,
          c.start_time,
          c.end_time,
          c.uuid,
          c.resources
        FROM checkpoints c
        WHERE c.trial_id = s.trial_id
          AND c.step_id = s.id
      ) r2
  ) AS checkpoint,
  (
    SELECT row_to_json(r3)
    FROM (
        SELECT s2.start_time,
          s2.end_time,
          'STATE_' || s2.state as state,
          s2.metrics->'avg_metrics' as metrics,
          s2.metrics->'num_inputs' as num_inputs
        FROM steps s2
        WHERE s2.trial_id = s.trial_id
          AND s2.id = s.id
      ) r3
  ) AS training
from steps s
WHERE s.trial_id = $1
ORDER BY s.id ASC
