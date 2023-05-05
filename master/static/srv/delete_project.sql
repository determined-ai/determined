DELETE FROM projects
WHERE id = $1
    AND NOT IMMUTABLE
RETURNING
    projects.id;

