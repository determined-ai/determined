WITH pins AS (
  SELECT id, workspace_id, created_at
  FROM workspace_pins
  WHERE user_id = $5
),
exp_count_by_project AS (
  SELECT COUNT(*) AS count, project_id FROM experiments
  GROUP BY project_id
)
SELECT w.id, w.name, w.archived, w.immutable, u.username, w.user_id,
(pins.id IS NOT NULL) AS pinned,
'WORKSPACE_STATE_' || w.state AS state, w.error_message,
(SELECT COUNT(*) FROM projects WHERE workspace_id = w.id) AS num_projects,
(SELECT SUM(count) FROM exp_count_by_project WHERE project_id IN
  (SELECT id FROM projects WHERE workspace_id = w.id)) AS num_experiments
FROM workspaces as w
LEFT JOIN users as u ON u.id = w.user_id
LEFT JOIN pins ON pins.workspace_id = w.id

WHERE ($1 = '' OR (u.username IN (SELECT unnest(string_to_array($1, ',')))))
AND ($2 = '' OR w.name ILIKE $2)
AND ($3 = '' OR w.archived = $3::BOOL)
AND ($4 = '' OR (pins.id IS NOT NULL) = $4::BOOL)
ORDER BY %s, pins.created_at DESC;
