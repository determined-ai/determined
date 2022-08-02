UPDATE workspaces
  SET state = 'DELETE_FAILED'
  WHERE id = $1
  AND state = 'DELETING'
RETURNING workspaces.id;
