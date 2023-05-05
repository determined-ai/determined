UPDATE
    models
SET
    archived = FALSE,
    last_updated_time = CURRENT_TIMESTAMP
WHERE
    name = $1
RETURNING
    id;

