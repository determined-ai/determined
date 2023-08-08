WITH mv AS (
    SELECT
        version,
        checkpoint_uuid,
        model_versions.id,
        creation_time,
        name,
        comment,
        metadata,
        labels,
        notes,
        username,
        user_id,
        last_updated_time
    FROM model_versions
    LEFT JOIN users ON users.id = model_versions.user_id
    WHERE model_id = $1 AND model_versions.version = $2
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
        count(mv.version) AS num_versions,
        m.workspace_id
    FROM models AS m
    JOIN users AS u ON u.id = m.user_id
    LEFT JOIN model_versions AS mv
        ON mv.model_id = m.id
    WHERE m.id = $1
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
    mv.notes,
    mv.name,
    mv.comment,
    mv.metadata,
    mv.username,
    mv.user_id,
    mv.last_updated_time
FROM c, m, mv;
