UPDATE projects
  SET state = 'DELETE_FAILED',
      error_message = $2
  WHERE id = $1
  AND state = 'DELETING'
RETURNING projects.id;
