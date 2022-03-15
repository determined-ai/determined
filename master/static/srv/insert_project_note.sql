UPDATE projects
SET notes = $2
WHERE id = $1
RETURNING id, name, description, notes, '' AS username;
