WITH p AS (
    INSERT INTO projects (name, description, workspace_id, user_id)
    VALUES ($1, $2, $3, $4)
    RETURNING id, name, description, archived, immutable, workspace_id, user_id
)

SELECT
    p.id,
    p.name,
    p.description,
    p.archived,
    p.immutable,
    p.workspace_id,
    p.user_id,
    u.username
FROM p
JOIN users u ON u.id = p.user_id;
