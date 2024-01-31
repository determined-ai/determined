DROP VIEW proto_checkpoints_view;
DROP VIEW checkpoints_view;
DROP VIEW trials;

ALTER TABLE public.runs
  ADD COLUMN request_id text,
  ADD COLUMN seed integer;

UPDATE runs SET
  request_id = t.request_id,
  seed = t.seed
FROM trials_v2 t
WHERE runs.id = t.run_id;

ALTER TABLE runs
  ALTER COLUMN seed SET NOT NULL,
  ALTER COLUMN seed SET DEFAULT 0;

DROP TABLE trials_v2;

CREATE VIEW trials AS
SELECT
  id AS id,

  -- metrics
  summary_metrics AS summary_metrics,
  summary_metrics_timestamp AS summary_metrics_timestamp,
  latest_validation_id AS latest_validation_id,
  total_batches AS total_batches,

  -- metadata fields
  state AS state,
  tags AS tags,
  external_run_id AS external_trial_id,
  restart_id AS run_id,
  last_activity AS last_activity,
  start_time AS start_time,
  end_time AS end_time,
  restarts AS restarts,
  -- project_id / owner_id will be propagated from experiments.

  -- run_hp_search_stuff
  hparams AS hparams,
  searcher_metric_value AS searcher_metric_value,
  searcher_metric_value_signed AS searcher_metric_value_signed,
  best_validation_id AS best_validation_id,

  -- run_checkpoint_stats
  checkpoint_size AS checkpoint_size,
  checkpoint_count AS checkpoint_count,

  -- trial_v2 table.
  request_id AS request_id,
  runner_state AS runner_state,
  seed AS seed,

  -- experiment_id will eventually be in the runs to run collection MTM.
  experiment_id AS experiment_id,

  -- warm_start_checkpoint_id will eventually be in the runs to checkpoint MTM.
  warm_start_checkpoint_id AS warm_start_checkpoint_id
FROM runs;

CREATE OR REPLACE VIEW checkpoints_view AS
    SELECT
        c.id AS id,
        c.uuid AS uuid,
        c.task_id,
        c.allocation_id,
        c.report_time,
        c.state,
        c.resources,
        c.metadata,
        t.id AS trial_id,
        e.id AS experiment_id,
        e.config AS experiment_config,
        t.hparams AS hparams,
        s.metrics AS training_metrics,
        v.metrics->'validation_metrics' AS validation_metrics,
        (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric,
        CAST(c.metadata->>'steps_completed' AS int) as steps_completed,
        c.size
    FROM checkpoints_v2 AS c
    LEFT JOIN trial_id_task_id AS task ON c.task_id = task.task_id
    LEFT JOIN trials AS t on t.id = task.trial_id
    LEFT JOIN experiments AS e on t.experiment_id = e.id
    LEFT JOIN raw_validations AS v on CAST(c.metadata->>'steps_completed' AS int) = v.total_batches and t.id = v.trial_id AND NOT v.archived
    LEFT JOIN raw_steps AS s on CAST(c.metadata->>'steps_completed' AS int) = s.total_batches and t.id = s.trial_id AND NOT s.archived;

CREATE OR REPLACE VIEW proto_checkpoints_view AS
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
    FROM checkpoints_view AS c;
