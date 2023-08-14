DELETE FROM workspace_pins
WHERE
    workspace_id = $1
    AND user_id = $2
RETURNING id;
