CREATE VIEW checkpoints_expanded AS
    SELECT
        c.id,
        c.trial_id,
        c.trial_run_id,
        c.total_batches,
        c.state,
        c.end_time,
        c.uuid,
        c.resources,
        c.metadata,
        c.framework,
        c.format,
        c.determined_version,

        e.config AS experiment_config,
        e.id AS experiment_id
        t.hparams,
        v.metrics AS validation_metrics,
        -- XXX where do I get the validation state from?
        v.metrics->'validation_metrics'->>(experiment_config->'searcher'->>'metric')::float8 AS searcher_metric
    FROM checkpoints AS c
    LEFT JOIN trials AS t ON c.trial_id = t.id
    LEFT JOIN experiments AS e ON t.experiment_id = e.id
    LEFT JOIN validations AS v ON c.total_batches = v.total_batches and c.trial_id = v.trial_id;
