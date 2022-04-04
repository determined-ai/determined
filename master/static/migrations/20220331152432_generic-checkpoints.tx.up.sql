CREATE TABLE public.checkpoints_v2 (
    id bigint NOT NULL DEFAULT nextval('public.checkpoints_id_seq'),
    uuid uuid NOT NULL UNIQUE,
    -- It is possible a checkpoint may be imported, in which case it would have no task or
    -- allocation ID. We make it non-null now to discourage relying on it so that feature is easier.
    task_id NULL text REFERENCES public.tasks(task_id),
    allocation_id NULL text REFERENCES public.allocations(allocation_id),
    report_time NOT NULL timestamp with time zone,
    state NOT NULL public.checkpoint_state NOT NULL,
    resources jsonb DEFAULT '{}'::jsonb,
    metadata jsonb DEFAULT '{}'::jsonb
);

-- checkpoints_view returns all checkpoints in the current format.
CREATE VIEW public.checkpoints_view AS
    SELECT
        c.uuid,
        t.task_id,
        c.end_time as report_time,
        c.state,
        c.resources,
        -- construct a metadata json from the user's metadata plus our training-specific fields that the
        -- TrialControllers inject when creating checkpoints.  Those values used to be "system" values,
        -- but since the release of Core API, the TrialControllers are no longer part of the system
        -- proper but are considered userspace tools.
        jsonb_build_object(
            'latest_batch', c.total_batches,
            'framework', c.framework,
            'determined_version', c.determined_version
        ) || COALESCE(c.metadata, '{}'::jsonb) as metadata,
        -- .Training substruct
        c.trial_id,
        t.experiment_id,
        e.config as experiment_config,
        t.hparams,
        s.metrics as training_metrics,
        'STATE_' || v.state AS validation_state,
        v.metrics as validation_metrics,
        (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric
        1 as checkpoint_version
    FROM raw_checkpoints AS c
    LEFT JOIN trials AS t on c.trial_id = t.id
    LEFT JOIN experiments AS  e on t.experiment_id = e.id
    LEFT JOIN validations AS v on c.total_batches = v.total_batches and c.trial_id = v.trial_id
    -- avoiding the steps view causes Postgres to not "Materialize" in this join.
    LEFT JOIN raw_steps AS s on c.total_batches = s.total_batches and c.trial_id = s.trial_id
    WHERE s.archived = false
    UNION
    SELECT
        c.uuid,
        c.task_id,
        c.report_time,
        c.state,
        c.resources,
        c.metadata,
        -- .Training substruct
        t.id as trial_id,
        t.experiment_id,
        e.config as experiment_config,
        t.hparams,
        s.metrics as training_metrics,
        'STATE_' || v.state AS validation_state,
        v.metrics as validation_metrics,
        (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric,
        2 as checkpoint_version
    FROM checkpoints_v2 AS c
    LEFT JOIN trials AS t on c.task_id = t.task_id
    LEFT JOIN experiments AS e on t.experiment_id = e.id
    LEFT JOIN validations AS v on c.metadata->>'latest_batch' = v.total_batches::text and t.id = v.trial_id
    -- avoiding the steps view causes Postgres to not "Materialize" in this join.
    LEFT JOIN raw_steps AS s on c.metadata->>'latest_batch' = s.total_batches::text and t.id = s.trial_id
    where s.archived = false;