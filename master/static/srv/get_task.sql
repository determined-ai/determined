SELECT tasks.task_id, state, is_ready
FROM tasks
  LEFT JOIN allocations
  ON allocations.task_id = tasks.task_id
WHERE tasks.task_id = $1
