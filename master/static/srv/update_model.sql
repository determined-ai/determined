UPDATE models SET description = $2, metadata = $3, labels = string_to_array($4, ','), last_updated_time = current_timestamp
WHERE id = $1
RETURNING name, description, metadata, array_to_json(labels) as labels, creation_time, last_updated_time
