SELECT
    j.job_id AS job,
    BOOL_OR(CASE WHEN a.state = 'PULLING' THEN true ELSE false END) AS pulling,
    BOOL_OR(
        CASE WHEN a.state = 'STARTING' THEN true ELSE false END
    ) AS starting,
    BOOL_OR(CASE WHEN a.state = 'RUNNING' THEN true ELSE false END) AS running
FROM jobs j
JOIN tasks t ON t.job_id = j.job_id
JOIN allocations a ON a.task_id = t.task_id
WHERE j.job_id IN (SELECT UNNEST(STRING_TO_ARRAY($1, ',')))
GROUP BY j.job_id
