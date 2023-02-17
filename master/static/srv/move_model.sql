UPDATE models SET workspace_id = $2
WHERE id = $1
RETURNING id;
