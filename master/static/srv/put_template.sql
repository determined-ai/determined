INSERT INTO templates (name, config, workspace_id)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET config=$2
RETURNING name, config, workspace_id
