SELECT
    c.uuid::text AS uuid,
    e.config AS experiment_config,
    e.id AS  experiment_id,
    t.id AS trial_id,
    t.hparams as hparams,
    c.total_batches AS batch_number,
    c.start_time AS start_time,
    c.end_time AS end_time,
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
LEFT JOIN validations v ON v.total_batches = c.total_batches AND v.trial_id = c.trial_id
JOIN trials t ON c.trial_id = t.id
JOIN experiments e ON t.experiment_id = e.id
WHERE c.uuid = $1
