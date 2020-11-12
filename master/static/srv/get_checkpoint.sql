SELECT
    c.uuid::text AS uuid,
    e.config AS experiment_config,
    e.id AS  experiment_id,
    t.id AS trial_id,
    t.hparams as hparams,
    s.prior_batches_processed + s.num_batches AS batch_number,
    s.start_time AS start_time,
    s.end_time AS end_time,
    c.resources AS resources,
    COALESCE(c.metadata, '{}') AS metadata,
    COALESCE(c.framework, '') as framework,
    COALESCE(c.format, '') as format,
    COALESCE(c.determined_version, '') as determined_version,
    v.metrics AS metrics,
    (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric,
    'STATE_' || v.state AS validation_state,
    'STATE_' || c.state AS state
FROM checkpoints c
JOIN steps s ON c.step_id = s.id AND c.trial_id = s.trial_id
LEFT JOIN validations v ON v.step_id = s.id AND v.trial_id = s.trial_id
JOIN trials t ON s.trial_id = t.id
JOIN experiments e ON t.experiment_id = e.id
WHERE c.uuid = $1
