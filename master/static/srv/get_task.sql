SELECT tasks.task_id, (SELECT coalesce(jsonb_agg(allo ORDER BY end_time DESC NULLS FIRST), '[]'::jsonb) FROM (
  SELECT allocation_id, task_id, 'STATE_' || state AS state, is_ready, start_time, end_time
  FROM allocations
  WHERE allocations.task_id = tasks.task_id
) allo) AS allocations
FROM tasks
WHERE tasks.task_id = $1;
