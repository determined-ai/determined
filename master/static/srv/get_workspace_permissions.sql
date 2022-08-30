WITH role_assignments AS (
  SELECT 1 AS group_id, 1 AS role_id, 1 AS scope_workspace_id
),
roles AS (
  SELECT 'edit_workspace' AS name
)
SELECT array_to_json(array_agg(name)) AS editor,
  array_to_json(array_agg(name)) AS viewer
FROM roles
  WHERE TRUE
  AND $1 > 0
  AND $2 > 0
  -- WHERE user_id = $1 AND workspace_id = $2
