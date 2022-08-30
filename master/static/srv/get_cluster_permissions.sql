WITH role_assignments AS (
  SELECT 1 AS group_id, 1 AS role_id
),
roles AS (
  SELECT 'view_user_permission' AS name
)
SELECT array_to_json(array_agg(name)) AS editor,
  array_to_json(array_agg(name)) AS viewer
FROM roles
  WHERE TRUE
  AND $1 > 0
  -- WHERE user_id = $1
