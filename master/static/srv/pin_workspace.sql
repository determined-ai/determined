INSERT INTO workspace_pins (workspace_id, user_id, created_at)
VALUES ($1, $2, NOW())
RETURNING id;
