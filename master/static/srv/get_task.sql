SELECT t.task_id,
  CONCAT('TASK_TYPE_',t.task_type) as task_type,
  t.start_time,
  t.end_time,
  (
    SELECT coalesce(
        jsonb_agg(
          allo
          ORDER BY end_time DESC NULLS FIRST
        ),
        '[]'::jsonb
      )
    FROM (
        SELECT allocation_id,
          task_id,
          is_ready,
          start_time,
          end_time,
          (
            CASE
              WHEN state IN ('PENDING', 'ASSIGNED') THEN 'STATE_QUEUED'
              ELSE 'STATE_' || state
            END
          ) AS state
        FROM allocations
        WHERE allocations.task_id = t.task_id
      ) allo
  ) AS allocations
FROM tasks t
WHERE t.task_id = $1;
