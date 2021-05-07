SELECT
    e.id AS id,
    /* TODO We could either do this and rely that we go through the apis that use this query
     (eg update cli) or migrate older experiments in database (one way). Look at model defaults */
    COALESCE(NULLIF(e.config->>'name', ''), NULLIF(e.config->>'description', ''), 'Experiment ' ||  e.id) AS name,
    e.config->>'description' AS description,
    e.config->'labels' AS labels,
    e.config->'resources'->>'resource_pool' as resource_pool,
    e.config->'searcher'->'name' as searcher_type,
    e.notes AS notes,
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
WHERE e.id = $1;
