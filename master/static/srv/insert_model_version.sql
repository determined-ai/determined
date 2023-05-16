WITH mv AS (
	INSERT INTO model_versions
		(
			model_id,
			version,
			checkpoint_uuid,
			name,
			comment,
			metadata,
			labels,
			notes,
			user_id,
			creation_time,
			last_updated_time
		)
	VALUES (
			$1,
			(SELECT COALESCE(max(version), 0) + 1 FROM model_versions WHERE model_id = $1),
			$2,
			$3,
			$4,
			$5,
			string_to_array($6, ','),
			$7,
			$8,
			current_timestamp,
			current_timestamp
		)
	RETURNING id, checkpoint_uuid, version, creation_time, name, comment, model_id, metadata, labels, user_id
),
u AS (
	SELECT username FROM users WHERE id = $8
),
m AS (
  SELECT m.id, m.name, m.description, m.notes, m.metadata, m.creation_time, m.last_updated_time, array_to_json(m.labels) AS labels, u.username, m.archived, COUNT(mv.version) as num_versions
  FROM models as m
  JOIN users as u ON u.id = m.user_id
  LEFT JOIN model_versions as mv
    ON mv.model_id = m.id
  WHERE m.id = $1
  GROUP BY m.id, u.id
),
c AS (
  SELECT *
  FROM proto_checkpoints_view c
  WHERE c.uuid IN (SELECT checkpoint_uuid::text FROM mv)
)
SELECT
    to_json(c) AS checkpoint,
    to_json(m) AS model,
		array_to_json(mv.labels) AS labels,
    mv.version, mv.id,
    mv.creation_time,
    mv.name, mv.comment, mv.metadata,
		u.username
    FROM c, mv, m, u
    WHERE c.uuid = mv.checkpoint_uuid::text;
