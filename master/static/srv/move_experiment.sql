WITH e AS (
  SELECT id, project_id FROM experiments
  WHERE id = $1
  AND NOT archived
),
p AS (
  SELECT projects.id, projects.workspace_id FROM projects, e
  WHERE projects.id = e.project_id
  AND NOT projects.archived
),
w AS (
  SELECT COUNT(*) FROM workspaces, p
  WHERE workspaces.id = p.workspace_id
  AND NOT workspaces.archived
)
UPDATE experiments SET project_id = $2
WHERE experiments.id = (SELECT id FROM e)
AND experiments.project_id = (SELECT id FROM p)
AND (SELECT count FROM w) > 0
RETURNING experiments.id;
