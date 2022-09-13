SELECT tasks.task_id, (SELECT coalesce(jsonb_agg(allo ORDER BY end_time DESC NULLS FIRST), '[]'::jsonb) FROM (
  SELECT allocation_id, task_id, is_ready, start_time, end_time,
  (CASE WHEN state IN ('PENDING', 'ASSIGNED')
    THEN 'STATE_QUEUED'
    ELSE 'STATE_' || state
  END) AS state
  FROM allocations
  WHERE allocations.task_id = tasks.task_id
) allo) AS allocations
FROM tasks
WHERE tasks.task_id = $1;
