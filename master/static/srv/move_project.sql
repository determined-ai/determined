UPDATE projects SET workspace_id = $2
WHERE id = $1
AND NOT immutable
RETURNING id;
