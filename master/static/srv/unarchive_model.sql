UPDATE models SET archived = false, last_updated_time = current_timestamp
WHERE id = $1
RETURNING id;
