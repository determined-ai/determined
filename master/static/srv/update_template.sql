UPDATE templates
SET config = $2
WHERE name = $1
RETURNING name,
  config,
  workspace_id
