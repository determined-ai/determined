UPDATE models SET description = $2, metadata = $3, last_updated_time = $4
WHERE id = $1
RETURNING name, description, metadata, creation_time, last_updated_time
