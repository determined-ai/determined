SELECT
    t.task_id AS task,
    BOOL_OR(
        CASE WHEN a.state = 'PULLING' THEN
            TRUE
        ELSE
            FALSE
        END) AS pulling,
    BOOL_OR(
        CASE WHEN a.state = 'STARTING' THEN
            TRUE
        ELSE
            FALSE
        END) AS starting,
    BOOL_OR(
        CASE WHEN a.state = 'RUNNING' THEN
            TRUE
        ELSE
            FALSE
        END) AS running
FROM
    tasks t
    JOIN allocations a ON a.task_id = t.task_id
WHERE
    t.task_id IN (
        SELECT
            unnest(string_to_array($1, ',')))
GROUP BY
    t.task_id
