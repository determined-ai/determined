SELECT tasks.task_id, (SELECT coalesce(jsonb_agg(allo), '[]'::jsonb) FROM (
  SELECT allocation_id, task_id, state, is_ready, start_time, end_time
  FROM allocations
  WHERE task_id = tasks.task_id
  ORDER BY end_time DESC NULLS FIRST
) allo) AS allocations
FROM tasks
WHERE tasks.task_id = $1;
