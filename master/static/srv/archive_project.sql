WITH p AS (
  SELECT id, workspace_id FROM projects
  WHERE id = $1
  AND NOT immutable
),
w AS (
  SELECT workspaces.id FROM p, workspaces
  WHERE workspaces.id = p.workspace_id
  AND NOT workspaces.archived
)
UPDATE projects SET archived = $2
  WHERE projects.id = $1
  AND NOT immutable
  AND workspace_id = (SELECT id FROM w)
RETURNING id;
