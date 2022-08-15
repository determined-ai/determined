WITH owned_workspaces AS (
  SELECT id
  FROM workspaces
  WHERE user_id = $2
)
UPDATE projects
  SET state = 'DELETING', error_message = NULL
  WHERE id = $1
  AND NOT immutable
  AND (user_id = $2 OR $3 IS TRUE
    OR workspace_id IN (SELECT id FROM owned_workspaces)
  )
RETURNING id;
