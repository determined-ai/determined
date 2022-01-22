UPDATE models SET archived = true, last_updated_time = current_timestamp
WHERE name = $1
RETURNING id;
