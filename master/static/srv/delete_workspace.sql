DELETE FROM workspaces
WHERE id = $1
AND (user_id = $2 OR $3 IS TRUE)
RETURNING workspaces.id;
