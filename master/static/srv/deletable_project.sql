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
)
SELECT id FROM projects
WHERE id IN (SELECT id FROM proj);
