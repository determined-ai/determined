SELECT
    t.task_id AS task,
    BOOL_OR(CASE WHEN a.state = 'PULLING' THEN true ELSE false END) AS pulling,
    BOOL_OR(
        CASE WHEN a.state = 'STARTING' THEN true ELSE false END
    ) AS starting,
    BOOL_OR(CASE WHEN a.state = 'RUNNING' THEN true ELSE false END) AS running
FROM tasks t
JOIN allocations a ON a.task_id = t.task_id
WHERE t.task_id IN (SELECT UNNEST(STRING_TO_ARRAY($1, ',')))
GROUP BY t.task_id
