WITH trial_ids AS (
    SELECT id
    FROM trials
    WHERE experiment_id = $1
)
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
    (SELECT json_agg(id) FROM trial_ids) AS trial_ids,
	(SELECT count(id) FROM trial_ids) AS num_trials
    e.project_id AS project_id
FROM
    experiments e
JOIN users u ON e.owner_id = u.id
WHERE e.id = $1
