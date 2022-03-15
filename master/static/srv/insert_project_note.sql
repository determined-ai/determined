WITH p AS (
  UPDATE projects
  SET notes = $2
  WHERE id = $1
  RETURNING id, name, description, notes, user_id
)
SELECT id, name, description, notes, u.username
FROM p
JOIN users u on u.id = p.user_id;
