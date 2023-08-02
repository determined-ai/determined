WITH m AS (
    INSERT INTO models (
        name,
        description,
        metadata,
        labels,
        notes,
        user_id,
        workspace_id,
        creation_time,
        last_updated_time
    )
    VALUES ($1, $2, $3, string_to_array($4, ','), $5, $6, $7, current_timestamp, current_timestamp)
    RETURNING name,
    description,
    notes,
    metadata,
    labels,
    user_id,
    workspace_id,
    creation_time,
    last_updated_time,
    id
)

SELECT
    m.name,
    m.description,
    m.workspace_id,
    m.notes,
    m.metadata,
    array_to_json(m.labels) AS labels,
    u.username,
    m.creation_time,
    m.last_updated_time,
    m.id
FROM m
JOIN users u ON u.id = m.user_id;
