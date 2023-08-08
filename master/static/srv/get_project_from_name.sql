SELECT
    p.id,
    p.name,
    p.description,
    p.immutable,
    (p.archived OR w.archived) AS archived
FROM projects p
JOIN workspaces w ON w.id = p.workspace_id
WHERE p.workspace_id = $1 AND p.name = $2;
