SELECT tasks.task_id AS TaskID,
  tasks.job_id AS JobID,
  SUM(case when allocations.state = 'RUNNING' then 1 else 0 end) AS NumRunning,
  COUNT(allocations) AS NumStarting
FROM tasks
JOIN allocations ON allocations.task_id = tasks.task_id
WHERE tasks.task_id IN (SELECT unnest(string_to_array($1, ','))) -- Trial match
  AND allocations.state IN ('PULLING', 'RUNNING', 'STARTING')
GROUP BY tasks.task_id, tasks.job_id;
