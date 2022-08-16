DELETE FROM workspaces
  WHERE id = $1
  AND NOT immutable
RETURNING id;
