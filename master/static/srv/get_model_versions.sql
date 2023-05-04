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
  SELECT m.id, m.name, m.description, m.notes, m.metadata, m.creation_time, m.last_updated_time, array_to_json(m.labels) AS labels, u.username, m.user_id, m.archived, COUNT(mv.version) as num_versions
  FROM models as m
  JOIN users as u ON u.id = m.user_id
  LEFT JOIN model_versions as mv
    ON mv.model_id = m.id
  WHERE m.id = $1
  GROUP BY m.id, u.id
),
cnv AS (
  SELECT c.id,
    c.uuid,
    c.task_id,
    c.allocation_id,
    c.report_time,
    c.state,
    c.resources,
    c.metadata,
    t.id AS trial_id,
    e.id AS experiment_id,
    e.config AS experiment_config,
    t.hparams,
    s.metrics AS training_metrics,
    v.metrics -> 'validation_metrics'::text AS validation_metrics,
    ((v.metrics -> 'validation_metrics'::text) ->> ((e.config -> 'searcher'::text) ->> 'metric'::text))::double precision AS searcher_metric,
    (c.metadata ->> 'steps_completed'::text)::integer AS steps_completed,
    2 AS checkpoint_version,
    c.size
   FROM checkpoints_v2 c
     LEFT JOIN trials t ON c.task_id = t.task_id
     LEFT JOIN experiments e ON t.experiment_id = e.id
     LEFT JOIN raw_validations v ON ((c.metadata ->> 'steps_completed'::text)::integer) = v.total_batches AND t.id = v.trial_id
     LEFT JOIN raw_steps s ON ((c.metadata ->> 'steps_completed'::text)::integer) = s.total_batches AND t.id = s.trial_id
  WHERE (s.archived IS NULL OR s.archived = false AND v.archived IS NULL OR v.archived = false)
    AND c.uuid IN (SELECT checkpoint_uuid FROM mv)
),
cov AS (
   SELECT c.id,
    c.uuid,
    t.task_id,
        CASE
            WHEN t.task_id IS NULL THEN NULL::text
            ELSE (t.task_id || '.'::text) || c.trial_run_id
        END AS allocation_id,
    c.end_time AS report_time,
    c.state,
    c.resources,
    jsonb_build_object('steps_completed', c.total_batches, 'framework', c.framework, 'format', c.format, 'determined_version', c.determined_version, 'experiment_config', e.config, 'hparams', t.hparams) || COALESCE(c.metadata, '{}'::jsonb) AS metadata,
    t.id AS trial_id,
    e.id AS experiment_id,
    e.config AS experiment_config,
    t.hparams,
    s.metrics AS training_metrics,
    v.metrics -> 'validation_metrics'::text AS validation_metrics,
    ((v.metrics -> 'validation_metrics'::text) ->> ((e.config -> 'searcher'::text) ->> 'metric'::text))::double precision AS searcher_metric,
    c.total_batches AS steps_completed,
    1 AS checkpoint_version,
    c.size
   FROM raw_checkpoints c
     LEFT JOIN trials t ON c.trial_id = t.id
     LEFT JOIN experiments e ON t.experiment_id = e.id
     LEFT JOIN raw_steps s ON s.trial_id = t.id AND s.trial_run_id = c.trial_run_id AND s.total_batches = c.total_batches
     LEFT JOIN raw_validations v ON v.trial_id = c.trial_id AND v.trial_run_id = c.trial_run_id AND v.total_batches = c.total_batches
  WHERE (s.archived IS NULL OR s.archived = false AND v.archived IS NULL OR v.archived = false)
  AND c.uuid IN (SELECT checkpoint_uuid FROM mv)
),
cv AS (
  SELECT cnv.id,
    cnv.uuid,
    cnv.task_id,
    cnv.allocation_id,
    cnv.report_time,
    cnv.state,
    cnv.resources,
    cnv.metadata,
    cnv.trial_id,
    cnv.experiment_id,
    cnv.experiment_config,
    cnv.hparams,
    cnv.training_metrics,
    cnv.validation_metrics,
    cnv.searcher_metric,
    cnv.steps_completed,
    cnv.checkpoint_version,
    cnv.size
   FROM cnv
  UNION ALL
  SELECT cov.id,
      cov.uuid,
      cov.task_id,
      cov.allocation_id,
      cov.report_time,
      cov.state,
      cov.resources,
      cov.metadata,
      cov.trial_id,
      cov.experiment_id,
      cov.experiment_config,
      cov.hparams,
      cov.training_metrics,
      cov.validation_metrics,
      cov.searcher_metric,
      cov.steps_completed,
      cov.checkpoint_version,
      cov.size
    FROM cov
),
pcv AS (
  SELECT c.uuid::text AS uuid,
    c.task_id,
    c.allocation_id,
    c.report_time,
    'STATE_'::text || c.state AS state,
    c.resources,
    c.metadata,
    jsonb_build_object('trial_id', c.trial_id, 'experiment_id', c.experiment_id, 'experiment_config', c.experiment_config, 'hparams', c.hparams, 'training_metrics', jsonb_build_object('avg_metrics', c.training_metrics -> 'avg_metrics'::text, 'batch_metrics', c.training_metrics -> 'batch_metrics'::text), 'validation_metrics', json_build_object('avg_metrics', c.validation_metrics), 'searcher_metric', c.searcher_metric) AS training
   FROM cv c
)
SELECT
    to_json(c) AS checkpoint,
    to_json(m) AS model,
    array_to_json(mv.labels) AS labels,
    mv.version, mv.id,
    mv.creation_time, mv.notes,
    mv.username, mv.user_id,
    mv.name, mv.comment, mv.metadata, mv.last_updated_time
    FROM pcv c, mv, m
    WHERE c.uuid = mv.checkpoint_uuid::text;
