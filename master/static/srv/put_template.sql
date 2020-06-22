INSERT INTO templates (name, config)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET config=$2
RETURNING name, config
