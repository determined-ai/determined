UPDATE
    workspaces
SET
    state = 'DELETING',
    error_message = NULL
WHERE
    id = $1
    AND NOT IMMUTABLE
RETURNING
    id;

