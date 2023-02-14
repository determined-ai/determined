UPDATE models SET workspace_id = $2
WHERE name = $1
RETURNING id;
