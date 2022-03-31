UPDATE projects SET workspace_id = $2
WHERE projects.id = $1
RETURNING projects.id;
