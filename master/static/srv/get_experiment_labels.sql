SELECT
    distinct e.config->'labels' AS labels
FROM
    experiments e
WHERE ($1 = 0) OR (project_id = $1)
