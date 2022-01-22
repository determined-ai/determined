UPDATE models SET name = $1, description = $2, notes = $3, metadata = $4, labels = string_to_array($5, ','), last_updated_time = current_timestamp
WHERE name = $1
RETURNING name, description, notes, metadata, array_to_json(labels) as labels, creation_time, last_updated_time
