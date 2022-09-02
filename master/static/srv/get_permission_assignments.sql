WITH role_assignments AS (
  SELECT 1 AS group_id, 1 AS role_id
),
workspace_assignments AS (
  SELECT id
  FROM workspaces
),
mocked AS (
  SELECT 'Foo Editor' AS name,
    array_agg(workspace_assignments.id) AS workspaces,
    FALSE as cluster
    FROM workspace_assignments
  UNION (
    SELECT 'Cluster Admin' AS name,
    ARRAY[]::integer[] AS workspaces,
    TRUE as cluster
  )
)
SELECT name, to_json(workspaces) AS workspaces, cluster
FROM mocked
WHERE $1 > 0
  -- WHERE user_id = $1
