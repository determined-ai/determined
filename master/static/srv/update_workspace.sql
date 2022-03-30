WITH w AS (
  UPDATE workspaces SET name = $2
  WHERE workspaces.id = $1
  RETURNING workspaces.*
),
u AS (
  SELECT username FROM users, w
  WHERE users.id = w.user_id
)
SELECT w.id, w.name, w.archived, w.immutable, u.username
FROM w, u;
