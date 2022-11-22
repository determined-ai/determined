-- TODO(DET-8692): Deduplicate code copied from `proto_checkpoints_view` without the performance hit
-- caused by `CAST(c.training->>'experiment_id' AS integer)` materializing more than we need.
SELECT
    c.uuid::text AS uuid,
    c.task_id,
    c.allocation_id,
    c.report_time as report_time,
    'STATE_' || c.state AS state,
    c.resources,
    c.metadata,
    -- Build a training substruct for protobuf.
    jsonb_build_object(
        'trial_id', c.trial_id,
        'experiment_id', c.experiment_id,
        'experiment_config', c.experiment_config,
        'hparams', c.hparams,
        -- construct training metrics from the untyped jsonb deterministically, since older
        -- versions may have old keys (e.g., num_inputs) and our unmarshaling is strict.
        'training_metrics', jsonb_build_object(
            'avg_metrics', c.training_metrics->'avg_metrics',
            'batch_metrics', c.training_metrics->'batch_metrics'
        ),
        'validation_metrics', json_build_object('avg_metrics', c.validation_metrics),
        'searcher_metric', c.searcher_metric
    ) AS training
FROM checkpoints_view AS c
WHERE c.experiment_id = $1
ORDER BY c.report_time DESC
