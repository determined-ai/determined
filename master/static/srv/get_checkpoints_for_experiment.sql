SELECT
    /* We need to cast uuid to text here otherwise the db.QueryProto method
       will try to deserialize the uuid as a []byte and parse it into json. */
    c.uuid::text AS uuid,
    'STATE_' || c.state AS state,
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
    'STATE_' || v.state AS validation_state,
    (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric
FROM checkpoints c
LEFT JOIN validations v ON v.total_batches = c.total_batches AND v.trial_id = c.trial_id
JOIN trials t ON c.trial_id = t.id
JOIN experiments e ON t.experiment_id = e.id
WHERE e.id = $1
ORDER BY start_time DESC
