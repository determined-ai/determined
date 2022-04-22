WITH w AS (
  SELECT id
  FROM workspaces
  WHERE id = $1
  AND NOT immutable
  AND (user_id = $2 OR $3 IS TRUE)
),
pins AS (
  DELETE FROM workspace_pins
  WHERE workspace_id IN (SELECT id FROM w)
)
DELETE FROM workspaces
WHERE id IN (SELECT id FROM w)
RETURNING id;
