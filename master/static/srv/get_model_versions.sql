WITH mv AS (
    SELECT
        version,
        checkpoint_uuid,
        creation_time,
        name,
        comment,
        model_versions.id,
        metadata,
        labels,
        notes,
        username,
        user_id,
        last_updated_time
    FROM model_versions
    LEFT JOIN users ON users.id = model_versions.user_id
    WHERE model_id = $1
),

m AS (
    SELECT
        m.id,
        m.name,
        m.description,
        m.notes,
        m.metadata,
        m.creation_time,
        m.last_updated_time,
        array_to_json(m.labels) AS labels,
        u.username,
        m.user_id,
        m.archived,
        count(mv.version) AS num_versions
    FROM models AS m
    JOIN users AS u ON u.id = m.user_id
    LEFT JOIN model_versions AS mv
        ON mv.model_id = m.id
    WHERE m.id = $1
    GROUP BY m.id, u.id
)

SELECT
    to_json(c) AS checkpoint,
    to_json(m) AS model,
    array_to_json(mv.labels) AS labels,
    mv.version,
    mv.id,
    mv.creation_time,
    mv.notes,
    mv.username,
    mv.user_id,
    mv.name,
    mv.comment,
    mv.metadata,
    mv.last_updated_time
FROM proto_checkpoints_view c, mv, m
WHERE c.uuid = mv.checkpoint_uuid::text;
