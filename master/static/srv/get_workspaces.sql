SELECT w.id, w.name, w.archived, w.immutable, u.username, w.user_id,
(pins.id IS NOT NULL) AS pinned, pins.created_at AS pinned_at,
'WORKSPACE_STATE_' || w.state AS state, w.error_message, w.default_compute_pool, w.default_aux_pool,
(CASE WHEN uid IS NOT NULL OR gid IS NOT NULL OR user_ IS NOT NULL OR group_ IS NOT NULL THEN
  jsonb_build_object('agent_uid', uid, 'agent_user', user_, 'agent_gid', gid, 'agent_group', group_)
  ELSE NULL END) AS agent_user_group,
COUNT(DISTINCT p.id) AS num_projects,
COUNT(DISTINCT e.id) AS num_experiments
FROM workspaces AS w
LEFT JOIN users AS u ON u.id = w.user_id
LEFT JOIN workspace_pins AS pins ON pins.workspace_id = w.id AND pins.user_id = $6
LEFT JOIN projects AS p ON p.workspace_id = w.id
LEFT JOIN experiments AS e ON e.project_id = p.id

WHERE ($1 = '' OR (u.username IN (SELECT unnest(string_to_array($1, ',')))))
AND ($2 = '' OR w.user_id IN (SELECT unnest(string_to_array($2, ',')::int [])))
AND ($3 = '' OR w.name ILIKE $3)
AND ($4 = '' OR w.archived = $4::BOOL)
AND ($5 = '' OR (pins.id IS NOT NULL) = $5::BOOL)

GROUP BY w.id, u.username, pins.id
ORDER BY %s, pins.created_at DESC;
