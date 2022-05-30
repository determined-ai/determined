WITH owned_workspaces AS (
  SELECT id
  FROM workspaces
  WHERE user_id = $2
),
proj AS (
  SELECT id FROM projects
  WHERE id = $1
  AND NOT immutable
  AND (user_id = $2 OR $3 IS TRUE
    OR workspace_id IN (SELECT id FROM owned_workspaces)
  )
),
exper AS (
  SELECT COUNT(*) AS count
  FROM experiments
  WHERE project_id IN (SELECT id FROM proj)
)
DELETE FROM projects
WHERE id IN (SELECT id FROM proj)
AND (SELECT count FROM exper) = 0
RETURNING projects.id;
