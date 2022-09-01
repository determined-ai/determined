SELECT
  j.job_id,
  BOOL_OR(CASE WHEN a.state IN ('PULLING', 'STARTING') THEN true ELSE false END) AS is_starting,
  BOOL_OR(CASE WHEN a.state = 'RUNNING' THEN true ELSE false END) AS is_running
FROM jobs j
JOIN tasks t ON t.job_id = j.job_id
JOIN allocations a ON a.task_id = t.task_id
WHERE j.job_id in (SELECT unnest(string_to_array($1, ',')))
GROUP BY j.job_id;
