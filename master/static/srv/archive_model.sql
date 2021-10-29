UPDATE models SET archived = true, last_updated_time = current_timestamp
WHERE id = $1
RETURNING id;
