WITH role_assignments AS (
  SELECT 1 AS group_id, 1 AS role_id
),
workspace_assignments AS (
  SELECT id FROM workspaces
),
editor AS (
  SELECT array_agg(id) AS workspaces
  FROM workspace_assignments
),
viewer AS (
  SELECT array_agg(id) AS workspaces
  FROM workspace_assignments
)
SELECT to_json(editor) AS editor,
  to_json(viewer) AS viewer
FROM role_assignments, editor, viewer
  WHERE TRUE
  AND $1 > 0
  -- WHERE user_id = $1
