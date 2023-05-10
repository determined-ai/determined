INSERT INTO templates (name, config, workspace_id)
VALUES ($1, $2, 1)
ON CONFLICT (name) DO UPDATE SET config=$2
RETURNING name, config
