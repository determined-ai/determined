UPDATE workspaces SET archived = $2
WHERE id = $1
AND NOT immutable
RETURNING id;
