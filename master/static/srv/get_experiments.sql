WITH page_info AS (
    SELECT public.page_info((
        -- Count the rows matching filters. Needed by pagination to show page numbers.
        SELECT COUNT(*) AS count
        FROM experiments e
        JOIN users u ON e.owner_id = u.id
        WHERE
            ($1 = '' OR e.state IN (SELECT unnest(string_to_array($1, ','))::experiment_state))
            AND ($2 = '' OR e.archived = $2::BOOL)
            AND ($3 = '' OR (u.username IN (SELECT unnest(string_to_array($3, ',')))))
            AND ($4 = '' OR e.owner_id IN (SELECT unnest(string_to_array($4, ',')::int [])))
            AND (
                    $5 = ''
                    OR string_to_array($5, ',') <@ ARRAY(SELECT jsonb_array_elements_text(
                        -- In the event labels were removed, if all were removed we insert null,
                        -- which previously broke this query.
                        CASE WHEN e.config->'labels'::text = 'null'
                        THEN NULL
                        ELSE e.config->'labels' END
                    ))
                )
            AND ($6 = '' OR (e.config->>'description') ILIKE  ('%%' || $6 || '%%'))
            AND ($7 = '' OR (e.config->>'name') ILIKE ('%%' || $7 || '%%'))
            AND ($8 = 0 OR e.project_id = $8)
    ), $9, $10) AS page_info
),
pj AS (
  SELECT projects.id, projects.name AS project_name, workspaces.name AS workspace_name
  FROM projects
  JOIN workspaces ON workspaces.id = projects.workspace_id
),
exps AS (
    SELECT
        e.id AS id,
        e.config->>'name' AS name,
        e.config->>'description' AS description,
        e.config->'labels' AS labels,
        e.config->'resources'->>'resource_pool' AS resource_pool,
        e.config->'searcher'->'name' as searcher_type,
        CASE
            WHEN NULLIF(e.notes, '') IS NULL THEN NULL
            ELSE 'omitted'
        END AS notes,
        e.start_time AS start_time,
        e.end_time AS end_time,
        'STATE_' || e.state AS state,
        (SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) AS num_trials,
        e.archived AS archived,
        e.progress AS progress,
        e.job_id AS job_id,
        e.parent_id AS forked_from,
        e.owner_id AS user_id,
        e.project_id AS project_id,
        u.username AS username,
        COALESCE(u.display_name, u.username) as display_name,
        pj.project_name,
        pj.workspace_name
    FROM experiments e
    JOIN users u ON e.owner_id = u.id
    JOIN pj ON e.project_id = pj.id
    WHERE
        ($1 = '' OR e.state IN (SELECT unnest(string_to_array($1, ','))::experiment_state))
        AND ($2 = '' OR e.archived = $2::BOOL)
        AND ($3 = '' OR (u.username IN (SELECT unnest(string_to_array($3, ',')))))
        AND ($4 = '' OR e.owner_id IN (SELECT unnest(string_to_array($4, ',')::int [])))
        AND (
                $5 = ''
                OR string_to_array($5, ',') <@ ARRAY(SELECT jsonb_array_elements_text(
                    -- In the event labels were removed, if all were removed we insert null,
                    -- which previously broke this query.
                    CASE WHEN e.config->'labels'::text = 'null'
                    THEN NULL
                    ELSE e.config->'labels' END
                ))
            )
        AND ($6 = '' OR (e.config->>'description') ILIKE  ('%%' || $6 || '%%'))
        AND ($7 = '' OR (e.config->>'name') ILIKE ('%%' || $7 || '%%'))
        AND ($8 = 0 OR e.project_id = $8)
    ORDER BY %s
    OFFSET (SELECT p.page_info->>'start_index' FROM page_info p)::bigint
    LIMIT (SELECT (p.page_info->>'end_index')::bigint - (p.page_info->>'start_index')::bigint FROM page_info p)
)
SELECT
    (SELECT coalesce(json_agg(exps), '[]'::json) FROM exps) AS experiments,
    (SELECT p.page_info FROM page_info p) AS pagination
