WITH p AS (
  INSERT INTO projects (name, description, user_id)
  VALUES ($1, $2, $3)
  RETURNING id, name, description, archived, user_id
)
SELECT p.id, p.name, p.description, p.archived, u.username
FROM p
JOIN users u on u.id = p.user_id;
