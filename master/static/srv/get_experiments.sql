SELECT
    e.id AS id,
    e.config->>'description' AS description,
    e.config->'labels' AS labels,
    e.config->'resources'->>'resource_pool' AS resource_pool,
    e.start_time AS start_time,
    e.end_time AS end_time,
    'STATE_' || e.state AS state,
    (SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) AS num_trials,
    e.archived AS archived,
    COALESCE(e.progress, 0) AS progress,
    u.username AS username
FROM
    experiments e
JOIN users u ON e.owner_id = u.id
