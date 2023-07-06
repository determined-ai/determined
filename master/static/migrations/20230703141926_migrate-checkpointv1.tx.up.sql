-- Endstate of checkpoint views / tables is
-- raw_checkpoints is left unmodified.
-- checkpoints_v2 has checkpoint_v2s and unarchived checkpoint_v1s.
-- checkpoints_view has all unarchived checkpoints pulling from checkpoints_v2.
-- proto_checkpoints_view is the proto version of checkpoints_view.

--ALTER TABLE public.checkpoints_v2
--    ADD COLUMN is_version_one boolean NOT NULL DEFAULT false; -- Allow migrate down.

-- TODO archived checkpoints_v1?

-- Insert unarchived checkpoints_v1 into checkpoints_v2. 
-- TODO using the view makes this easy but we join on steps and validations, which isn't ideal
--     we might need this join anyway so
--     I don't think we can get around it just by using the raw table.
INSERT INTO public.checkpoints_v2 (
    uuid,
    task_id,
    allocation_id,
    report_time,
    state,
    resources,
    metadata,
    size
)
SELECT
    c.uuid,
    c.task_id,
    CASE -- TODO is this behaviour okay? Or should we backfill and insert into allocations.
        WHEN a.allocation_id IS NULL THEN NULL
        ELSE c.allocation_id
    END,
    c.report_time,
    c.state,
    c.resources,
    c.metadata,
    c.size
FROM public.checkpoints_old_view c
LEFT JOIN public.allocations a ON c.allocation_id = a.allocation_id;

-- Note we just leave checkpoints_v1 data so we can reverse this migration.


-- Delete data that was migrated over.
--DELETE FROM public.checkpoints WHERE uuid IN (
--    SELECT uuid FROM public.checkpoints_v2 WHERE is_version_one
--);

DROP VIEW public.proto_checkpoints_view;
DROP VIEW public.checkpoints_view;
DROP VIEW public.checkpoints_old_view;
DROP VIEW public.checkpoints_new_view;
DROP VIEW public.checkpoints;

CREATE OR REPLACE VIEW public.checkpoints_view AS
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
        -- Remove checkpoint version since it doesn't make sense anymore. 2 AS checkpoint_version,
        c.size
    FROM checkpoints_v2 AS c
    LEFT JOIN trials AS t on c.task_id = t.task_id
    LEFT JOIN experiments AS e on t.experiment_id = e.id
    LEFT JOIN raw_validations AS v on CAST(c.metadata->>'steps_completed' AS int) = v.total_batches and t.id = v.trial_id
    LEFT JOIN raw_steps AS s on CAST(c.metadata->>'steps_completed' AS int) = s.total_batches and t.id = s.trial_id
    -- avoiding the steps view causes Postgres to not "Materialize" in this join.
    WHERE s.archived IS NULL OR s.archived = false
      AND v.archived IS NULL OR v.archived = false;

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
