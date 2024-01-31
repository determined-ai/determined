-- Create run_checkpoints MTM.
CREATE TABLE run_checkpoints (
  run_id integer REFERENCES runs(id) ON DELETE CASCADE NOT NULL,
  checkpoint_id uuid UNIQUE REFERENCES checkpoints_v2(uuid) ON DELETE CASCADE NOT NULL UNIQUE,
  PRIMARY KEY(run_id, checkpoint_id)
);

INSERT INTO run_checkpoints(run_id, checkpoint_id)
SELECT t.trial_id AS run_id, uuid AS checkpoint_id
FROM checkpoints_v2 c
JOIN trial_id_task_id t ON c.task_id = t.task_id;

CREATE INDEX idx_checkpoint_id_run_id ON run_checkpoints USING btree (checkpoint_id, run_id);

-- Update checkpoints_v2 and views.
DROP VIEW proto_checkpoints_view;
DROP VIEW checkpoints_view;

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
        r.id AS trial_id,
        e.id AS experiment_id,
        e.config AS experiment_config,
        r.hparams AS hparams,
        s.metrics AS training_metrics,
        v.metrics->'validation_metrics' AS validation_metrics,
        (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric,
        CAST(c.metadata->>'steps_completed' AS int) as steps_completed,
        c.size
    FROM checkpoints_v2 AS c
    LEFT JOIN run_checkpoints AS rc on rc.checkpoint_id = c.uuid
    LEFT JOIN runs AS r on r.id = rc.run_id
    LEFT JOIN experiments AS e on r.experiment_id = e.id
    LEFT JOIN raw_validations AS v on CAST(c.metadata->>'steps_completed' AS int) = v.total_batches and r.id = v.trial_id AND NOT v.archived
    LEFT JOIN raw_steps AS s on CAST(c.metadata->>'steps_completed' AS int) = s.total_batches and r.id = s.trial_id AND NOT s.archived;

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
