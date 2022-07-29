SELECT id
  FROM workspaces
  WHERE id = $1
  AND NOT immutable
  AND (user_id = $2 OR $3 IS TRUE)
;
