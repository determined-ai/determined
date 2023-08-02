SELECT
    w.id,
    w.name,
    w.archived,
    w.immutable
FROM workspaces w
WHERE w.name = $1;
