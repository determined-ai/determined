UPDATE
    models
SET
    archived = TRUE,
    last_updated_time = CURRENT_TIMESTAMP
WHERE
    name = $1
RETURNING
    id;

