WITH mv AS (
  SELECT version, checkpoint_uuid
    FROM model_versions
    WHERE model_name = $1 AND version = $2
),
m AS (
  Select * FROM models WHERE name = $1
),
c AS (
  SELECT
    c.uuid::text AS uuid,
    e.config AS experiment_config,
    e.id AS  experiment_id,
    t.id AS trial_id,
    t.hparams as hparams,
    s.total_batches AS batch_number,
    s.start_time AS start_time,
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
  WHERE c.uuid = (SELECT checkpoint_uuid FROM mv)
)
SELECT
    to_json(c) AS checkpoint,
    to_json(m) AS model,
    version AS version,
    creation_time
    FROM c, m, mv
