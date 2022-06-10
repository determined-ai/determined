WITH w AS (
  SELECT id
  FROM workspaces
  WHERE id = $1
  AND NOT immutable
  AND (user_id = $2 OR $3 IS TRUE)
),
proj AS (
  SELECT id FROM projects
  WHERE workspace_id IN (SELECT id FROM w)
),
exper AS (
  SELECT COUNT(*) AS count
  FROM experiments
  WHERE project_id IN (SELECT id FROM proj)
),
del_p AS (
  DELETE FROM projects
  WHERE id IN (SELECT id FROM proj)
  AND (SELECT count FROM exper) = 0
)
DELETE FROM workspaces
WHERE id IN (SELECT id FROM w)
AND (SELECT count FROM exper) = 0
RETURNING id;
