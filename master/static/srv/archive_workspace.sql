WITH w AS (
  SELECT id
  FROM workspaces
  WHERE id = $1
  AND NOT immutable
)
UPDATE workspaces SET archived = $2
WHERE id = (SELECT id FROM w)
RETURNING id;
