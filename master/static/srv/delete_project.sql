DELETE FROM projects
  WHERE id = $1
  AND NOT immutable
RETURNING projects.id;
