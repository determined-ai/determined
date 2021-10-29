WITH mv AS (
	INSERT INTO model_versions
		(
			model_id,
			version,
			checkpoint_uuid,
			metadata,
			labels,
			notes,
			creation_time,
			last_updated_time
		)
	VALUES (
			$1,
			(SELECT COALESCE(max(version), 0) + 1 FROM model_versions WHERE model_id = $1),
			$2,
			$3,
			string_to_array($4, ','),
			$5,
			current_timestamp,
			current_timestamp
		)
	RETURNING id, checkpoint_uuid, version, creation_time, name, comment, model_id, metadata, labels
),
m AS (
  SELECT m.id, m.name, m.description, m.metadata, m.creation_time, m.last_updated_time, array_to_json(m.labels) AS labels, u.username, m.archived, COUNT(mv.version) as num_versions
  FROM models as m
  JOIN users as u ON u.id = m.user_id
  LEFT JOIN model_versions as mv
    ON mv.model_id = m.id
  WHERE m.id = $1
  GROUP BY m.id, u.id
),
c AS (
  SELECT
    c.uuid::text AS uuid,
    e.config AS experiment_config,
    e.id AS  experiment_id,
    t.id AS trial_id,
    t.hparams as hparams,
    s.total_batches AS batch_number,
    s.end_time AS end_time,
    c.resources AS resources,
    COALESCE(c.metadata, '{}') AS metadata,
    COALESCE(c.framework, '') as framework,
    COALESCE(c.format, '') as format,
    COALESCE(c.determined_version, '') as determined_version,
    v.metrics AS metrics,
    'STATE_' || v.state AS validation_state,
    'STATE_' || c.state AS state
  FROM checkpoints c
  JOIN steps s ON c.total_batches = s.total_batches AND c.trial_id = s.trial_id
  LEFT JOIN validations v ON v.total_batches = s.total_batches AND v.trial_id = s.trial_id
  JOIN trials t ON s.trial_id = t.id
  JOIN experiments e ON t.experiment_id = e.id
  WHERE c.uuid IN (SELECT checkpoint_uuid FROM mv)
)
SELECT
    to_json(c) AS checkpoint,
    to_json(m) AS model,
		array_to_json(mv.labels) AS labels,
    mv.version, mv.id,
    mv.creation_time,
    mv.name, mv.comment, mv.metadata
    FROM c, mv, m
    WHERE c.uuid = mv.checkpoint_uuid::text;
