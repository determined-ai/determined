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
                OR string_to_array($4, ',') <@ ARRAY(SELECT jsonb_array_elements_text(
                    -- In the event labels were removed, if all were removed we insert null,
                    -- which previously broke this query.
                    CASE WHEN e.config->'labels'::text = 'null'
                    THEN NULL
                    ELSE e.config->'labels' END
                ))
            )
        AND ($5 = '' OR POSITION($5 IN (e.config->>'description')) > 0)
), page_info AS (
    SELECT public.page_info((SELECT COUNT(*) AS count FROM filtered_exps), $6, $7) AS page_info
)
SELECT
   (SELECT coalesce(json_agg(paginated_exps), '[]'::json) FROM (
        SELECT * FROM filtered_exps
        ORDER BY %s
        OFFSET (SELECT p.page_info->>'start_index' FROM page_info p)::bigint
        LIMIT (SELECT (p.page_info->>'end_index')::bigint - (p.page_info->>'start_index')::bigint FROM page_info p)
    ) AS paginated_exps) AS experiments,
    (SELECT p.page_info FROM page_info p) AS pagination

