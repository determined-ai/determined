SELECT
    e.id AS id,
    e.config->>'description' AS description,
    e.config->'labels' AS labels,
    e.start_time AS startTime,
    e.end_time AS endTime,
    e.state AS state,
    (SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) AS numTrials,
    e.archived AS archived,
    COALESCE(e.progress, 0) AS progress,
    u.username AS username
FROM
    experiments e
JOIN users u ON e.owner_id = u.id