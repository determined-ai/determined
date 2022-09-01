WITH role_assignments AS (
  SELECT 1 AS group_id, 1 AS role_id
),
workspace_assignments AS (
  SELECT id
  FROM workspaces
)
SELECT 'Foo Editor' AS name,
  to_json(array_agg(workspace_assignments.id)) AS workspaces
FROM role_assignments, workspace_assignments
  WHERE TRUE
  AND $1 > 0
  -- WHERE user_id = $1
