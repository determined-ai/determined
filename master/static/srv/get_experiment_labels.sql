SELECT
    distinct e.config->'labels' AS labels
FROM
    experiments e
