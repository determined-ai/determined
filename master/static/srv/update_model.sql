UPDATE models SET name = $2, description = $3, notes = $4, metadata = $5, labels = string_to_array($6, ','), workspace_id = $7, last_updated_time = current_timestamp
WHERE id = $1
RETURNING name, description, notes, metadata, array_to_json(labels) as labels, creation_time, last_updated_time
