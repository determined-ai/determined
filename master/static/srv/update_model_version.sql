WITH mv AS (
    UPDATE model_versions
    SET
        name = $3,
        comment = $4,
        notes = $5,
        metadata = $6,
        labels = string_to_array($7, ','),
        last_updated_time = current_timestamp
    WHERE id = $1
    RETURNING id,
    version,
    checkpoint_uuid,
    model_id,
    last_updated_time,
    creation_time,
    name,
    comment,
    notes,
    labels,
    metadata
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
        m.archived,
        count(mv.version) AS num_versions
    FROM models AS m
    JOIN users AS u ON u.id = m.user_id
    LEFT JOIN model_versions AS mv
        ON mv.model_id = m.id
    WHERE m.id = $2
    GROUP BY m.id, u.id
),

c AS (
    SELECT *
    FROM proto_checkpoints_view c
    WHERE c.uuid = (SELECT checkpoint_uuid::text FROM mv)
)

SELECT
    to_json(c) AS checkpoint,
    to_json(m) AS model,
    array_to_json(mv.labels) AS labels,
    mv.version,
    mv.id,
    mv.creation_time,
    mv.name,
    mv.comment,
    mv.notes,
    mv.metadata
FROM c, m, mv;
