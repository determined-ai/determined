CREATE VIEW bun_checkpoints_view AS
SELECT c.uuid::text                          AS uuid,
       c.task_id,
       c.allocation_id,
       c.report_time,
       'STATE_'::text || c.state             AS state,
       c.resources,
       c.metadata,
       jsonb_build_object('trial_id', c.trial_id, 'experiment_id', c.experiment_id, 'experiment_config',
                          c.experiment_config, 'hparams', c.hparams, 'training_metrics',
                          jsonb_build_object('avg_metrics', c.training_metrics -> 'avg_metrics'::text, 'batch_metrics',
                                             c.training_metrics -> 'batch_metrics'::text), 'validation_metrics',
                          json_build_object('avg_metrics', c.validation_metrics), 'searcher_metric',
                          c.searcher_metric) AS training,
        c.experiment_id,
        c.trial_id,
        c.state AS state_enum
FROM checkpoints_view c;

