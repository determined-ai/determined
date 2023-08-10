ALTER TABLE public.experiments
    ALTER COLUMN checkpoint_size DROP NOT NULL,
    ALTER COLUMN checkpoint_size SET DEFAULT NULL,
    ALTER COLUMN checkpoint_count DROP NOT NULL,
    ALTER COLUMN checkpoint_count SET DEFAULT NULL;

ALTER TABLE public.trials
    ALTER COLUMN checkpoint_size DROP NOT NULL,
    ALTER COLUMN checkpoint_size SET DEFAULT NULL,
    ALTER COLUMN checkpoint_count DROP NOT NULL,
    ALTER COLUMN checkpoint_count SET DEFAULT NULL;

DROP VIEW public.proto_checkpoints_view;
DROP VIEW public.checkpoints_view;
DROP VIEW public.checkpoints_old_view;
DROP VIEW public.checkpoints_new_view;
DROP VIEW public.checkpoints;

ALTER TABLE public.checkpoints_v2
    DROP COLUMN size;

ALTER TABLE public.raw_checkpoints
    DROP COLUMN size;

-- Copied from /migrations/20220504154053_add-checkpoints-v2-steps-completed-index.tx.up.sql
CREATE OR REPLACE VIEW public.checkpoints_old_view AS
    SELECT
        c.id AS id,
        c.uuid AS uuid,
        t.task_id,
        CASE
        WHEN t.task_id is NULL THEN
            NULL
        ELSE
            t.task_id || '.' || c.trial_run_id
        END allocation_id,
        c.end_time as report_time,
        c.state,
        c.resources,
        -- construct a metadata json from the user's metadata plus our training-specific fields that the
        -- TrialControllers inject when creating checkpoints.  Those values used to be "system" values,
        -- but since the release of Core API, the TrialControllers are no longer part of the system
        -- proper but are considered userspace tools.
        jsonb_build_object(
            'steps_completed', c.total_batches,
            'framework', c.framework,
            'format', c.format,
            'determined_version', c.determined_version,
            'experiment_config', e.config,
            'hparams', t.hparams
        ) || COALESCE(c.metadata, '{}'::jsonb) AS metadata,
        t.id AS trial_id,
        e.id AS experiment_id,
        e.config AS experiment_config,
        t.hparams AS hparams,
        s.metrics AS training_metrics,
        v.metrics->'validation_metrics' AS validation_metrics,
        (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric,
        c.total_batches as steps_completed,
        1 as checkpoint_version
    FROM raw_checkpoints AS c
    LEFT JOIN trials AS t on c.trial_id = t.id
    LEFT JOIN experiments AS e on t.experiment_id = e.id
    LEFT JOIN raw_steps AS s ON (
        -- Hint to the query planner to use the matching index.
        s.trial_id = t.id
        AND s.trial_run_id = c.trial_run_id
        AND s.total_batches = c.total_batches
    )
    LEFT JOIN raw_validations AS v ON (
        -- Hint to the query planner to use the matching index.
        v.trial_id = c.trial_id
        AND v.trial_run_id = c.trial_run_id
        AND v.total_batches = c.total_batches
    )
    -- Avoiding the steps and validation view causes Postgres to not "Materialize" in this join.
    WHERE s.archived IS NULL OR s.archived = false
      AND v.archived IS NULL OR v.archived = false;

-- Copied from /migrations/20220504154053_add-checkpoints-v2-steps-completed-index.tx.up.sql
CREATE OR REPLACE VIEW public.checkpoints_new_view AS
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
        2 AS checkpoint_version
    FROM checkpoints_v2 AS c
    LEFT JOIN trials AS t on c.task_id = t.task_id
    LEFT JOIN experiments AS e on t.experiment_id = e.id
    LEFT JOIN raw_validations AS v on CAST(c.metadata->>'steps_completed' AS int) = v.total_batches and t.id = v.trial_id
    LEFT JOIN raw_steps AS s on CAST(c.metadata->>'steps_completed' AS int) = s.total_batches and t.id = s.trial_id
    -- avoiding the steps view causes Postgres to not "Materialize" in this join.
    WHERE s.archived IS NULL OR s.archived = false
      AND v.archived IS NULL OR v.archived = false;

-- Copied from /migrations/20220504154053_add-checkpoints-v2-steps-completed-index.tx.up.sql
CREATE OR REPLACE VIEW public.checkpoints_view AS
    SELECT * FROM checkpoints_new_view
    UNION ALL
    SELECT * FROM checkpoints_old_view;

-- Copied from /migrations/20220502180510_generic-checkpoints.tx.up.sql
CREATE OR REPLACE VIEW public.proto_checkpoints_view AS
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

CREATE OR REPLACE VIEW checkpoints AS
    SELECT * FROM raw_checkpoints WHERE NOT archived;
