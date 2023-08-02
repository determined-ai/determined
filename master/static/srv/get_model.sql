SELECT
    m.id,
    m.name,
    m.description,
    m.notes,
    m.metadata,
    m.creation_time,
    m.last_updated_time,
    array_to_json(m.labels) AS labels,
    m.user_id,
    u.username,
    m.workspace_id,
    m.archived,
    count(mv.version) AS num_versions
FROM models AS m
LEFT JOIN model_versions AS mv
    ON mv.model_id = m.id
LEFT JOIN users AS u ON u.id = m.user_id
WHERE m.name = $1
GROUP BY m.id, u.id;
