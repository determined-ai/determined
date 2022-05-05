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
),
proj AS (
  SELECT id FROM projects
  WHERE workspace_id IN (SELECT id FROM w)
),
exper AS (
  UPDATE experiments SET project_id = 1
  WHERE project_id IN (SELECT id FROM proj)
),
del_proj AS (
  DELETE FROM projects
  WHERE id IN (SELECT id FROM proj)
)
DELETE FROM workspaces
WHERE id IN (SELECT id FROM w)
RETURNING id;
