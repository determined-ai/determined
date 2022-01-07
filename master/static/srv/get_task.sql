WITH allo AS (
  SELECT task_id, state, is_ready, start_time, end_time
  FROM allocations
  WHERE task_id = $1
  ORDER BY end_time DESC NULLS FIRST
)
SELECT tasks.task_id, json_build_array(allo) AS allocations
FROM tasks
LEFT JOIN allo ON tasks.task_id = allo.task_id
WHERE tasks.task_id = $1
GROUP BY tasks.task_id, allo.*;
