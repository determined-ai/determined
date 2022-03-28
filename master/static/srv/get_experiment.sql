SELECT
    e.id AS id,
    e.config->>'name' AS name,
    e.config->>'description' AS description,
    e.config->'labels' AS labels,
    e.config->'resources'->>'resource_pool' as resource_pool,
    e.config->'searcher'->'name' as searcher_type,
    e.notes AS notes,
    e.start_time AS start_time,
    e.end_time AS end_time,
    'STATE_' || e.state AS state,
    e.archived AS archived,
    e.progress AS progress,
    e.job_id AS job_id,
    e.parent_id AS forked_from,
    e.owner_id AS user_id,
    u.username AS username,
	array_to_json(ARRAY_AGG(t.id) filter (where t.id is not null)) AS trial_ids,
	COUNT(t.id) AS num_trials
FROM
    experiments e
JOIN users u ON e.owner_id = u.id
LEFT JOIN trials t ON e.id = t.experiment_id
WHERE e.id = $1
GROUP by e.id, u.username