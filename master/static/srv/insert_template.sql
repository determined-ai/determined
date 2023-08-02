INSERT INTO templates (name, config, workspace_id)
VALUES ($1, $2, $3)
RETURNING name, config, workspace_id
