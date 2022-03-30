WITH w AS (
  INSERT INTO workspaces (name, user_id)
  VALUES ($1, $2)
  RETURNING id, name, archived, immutable, user_id
)
SELECT w.id, w.name, w.archived, w.immutable, u.username
FROM w
JOIN users u on u.id = w.user_id;
