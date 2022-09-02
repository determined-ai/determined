WITH cluster_permissions AS (SELECT
  1 AS id,
  'view_user_permission' AS name,
  True AS global_only,
  False AS workspace_only
),
roles AS (
  SELECT
    1 AS id,
    'Cluster Admin' AS name
  UNION SELECT
    2 as id,
    'Foo Editor' AS name
)
SELECT roles.id, roles.name,
  to_json(array_agg(cluster_permissions)) AS permissions
FROM roles
  JOIN cluster_permissions ON TRUE
  WHERE TRUE
  AND $1 > 0
GROUP BY roles.id, roles.name
  -- WHERE user_id = $1
