WITH wp AS (
  SELECT id, workspace_id, created_at
  FROM workspace_pins
  WHERE user_id = $5
)
SELECT w.id, w.name, w.archived, w.immutable, u.username, w.user_id,
(wp.id IS NOT NULL) AS pinned,
(SELECT COUNT(*) FROM projects WHERE workspace_id = w.id) AS num_projects

FROM workspaces as w
LEFT JOIN users as u ON u.id = w.user_id
LEFT JOIN wp ON wp.workspace_id = w.id

WHERE ($1 = '' OR (u.username IN (SELECT unnest(string_to_array($1, ',')))))
AND ($2 = '' OR w.name ILIKE $2)
AND ($3 = '' OR w.archived = $3::BOOL)
AND ($4 = '' OR (wp.id IS NOT NULL) = $4::BOOL)
ORDER BY %s, wp.created_at DESC;
