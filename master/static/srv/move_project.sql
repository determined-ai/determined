WITH p AS (
  SELECT id, workspace_id FROM projects
  WHERE id = $1
  AND NOT immutable
),
origin_w AS (
  SELECT workspaces.id FROM workspaces, p
  WHERE workspaces.id = p.workspace_id
  AND NOT workspaces.archived
)
UPDATE projects SET workspace_id = $2
WHERE id = (SELECT id FROM p)
AND workspace_id = (SELECT id FROM origin_w)
RETURNING id;
