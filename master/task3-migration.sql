-- XXX We'll need to key checkpoints by UUID and not by ID;
-- when we see a checkpoint id we'll have to assume its old, and when we see
-- a uuid we should not assume if it is old or new.
-- XXX: foreign key to new_checkpoints?
alter table trials add column warm_start_checkpoint_uuid uuid;

create table public.checkpoints_new (
    -- XXX: why not just use uuid as the primary key?
    id bigserial PRIMARY KEY,
    uuid uuid NOT NULL UNIQUE,
    -- XXX: foreign key to tasks table
    task_id text,
    -- XXX: add this later?  Or what?
    allocation_id text,
    -- XXX: why on earth do we allow time zone?
    report_time timestamp with time zone,
    -- XXX: default these?
    state public.checkpoint_state NOT NULL,
    resources jsonb DEFAULT '{}'::jsonb,
    metadata jsonb DEFAULT '{}'::jsonb
);

-- This is fairly well-optimized already, mostly by @stokc
CREATE VIEW checkpoints_expanded AS
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
        v.metrics as validation_metrics
    FROM raw_checkpoints AS c
    LEFT JOIN trials AS t on c.trial_id = t.id
    LEFT JOIN experiments AS e on t.experiment_id = e.id
    LEFT JOIN validations AS v on c.total_batches = v.total_batches and c.trial_id = v.trial_id
    -- avoiding the steps view causes Postgres to not "Materialize" in this join.
    LEFT JOIN raw_steps AS s on c.total_batches = s.total_batches and c.trial_id = s.trial_id
    where s.archived = false
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
        v.metrics as validation_metrics
    FROM checkpoints_new AS c
    LEFT JOIN trials AS t on c.task_id = t.task_id
    LEFT JOIN experiments AS e on t.experiment_id = e.id
    LEFT JOIN validations AS v on c.metadata->>'latest_batch' = v.total_batches::text and t.id = v.trial_id
    -- avoiding the steps view causes Postgres to not "Materialize" in this join.
    LEFT JOIN raw_steps AS s on c.metadata->>'latest_batch' = s.total_batches::text and t.id = s.trial_id
    where s.archived = false;
