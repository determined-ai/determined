UPDATE projects
  SET state = 'DELETING', error_message = NULL
  WHERE id = $1
  AND NOT immutable
RETURNING id;
