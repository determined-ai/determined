WITH filtered_exps AS (
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
    FROM experiments e
    JOIN users u ON e.owner_id = u.id
    WHERE
        ($1 = '' OR e.state IN (SELECT unnest(string_to_array($1, ','))::experiment_state))
        AND ($2 = '' OR e.archived = $2::BOOL)
        AND ($3 = '' OR (u.username IN (SELECT unnest(string_to_array($3, ',')))))
        AND (
                $4 = ''
                OR string_to_array($4, ',') <@ ARRAY(SELECT jsonb_array_elements_text(e.config->'labels'))
            )
        AND ($5 = '' OR POSITION($5 IN (e.config->>'description')) > 0)
    )
SELECT
    (SELECT COUNT(*) FROM filtered_exps) as count,
    (SELECT json_agg(paginated_exps) FROM (
        SELECT * FROM filtered_exps
        ORDER BY %s
        OFFSET $6
        LIMIT $7
    ) AS paginated_exps) AS experiments
