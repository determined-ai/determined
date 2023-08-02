WITH p AS (
    SELECT workspace_id
    FROM projects
    WHERE id = $1
)

SELECT
    w.id,
    w.name,
    w.archived,
    w.immutable
FROM p, workspaces w
WHERE w.id = p.workspace_id;
