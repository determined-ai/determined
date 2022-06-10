INSERT INTO workspace_pins (workspace_id, user_id)
VALUES ($1, $2)
RETURNING id;
