DELETE FROM projects
  WHERE id = $1
  AND NOT immutable
  AND (user_id = $2 OR $3 IS TRUE
    OR workspace_id IN (SELECT id FROM owned_workspaces)
  )
RETURNING projects.id;