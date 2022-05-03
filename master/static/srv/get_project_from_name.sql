SELECT p.id, p.name, p.description, p.immutable
FROM projects p
WHERE p.workspace_id = $1 AND p.name = $2;
