DROP TABLE public.trial_snapshots;

DROP TABLE public.runs;

DROP TYPE public.run_type;

ALTER TABLE public.task_sessions RENAME TO allocation_sessions;
TRUNCATE TABLE public.allocation_sessions;
ALTER TABLE public.allocation_sessions ALTER COLUMN task_id TYPE text USING task_id::text;
ALTER TABLE public.allocation_sessions RENAME COLUMN task_id TO allocation_id;
ALTER TABLE public.allocation_sessions ADD CONSTRAINT allocation_sessions_sessions_allocation_id_uniq UNIQUE (allocation_id);

ALTER TYPE public.experiment_state RENAME TO _experiment_state;
CREATE TYPE public.experiment_state AS ENUM (
    'ACTIVE',
    'PAUSED',
    'STOPPING_CANCELED',
    'STOPPING_KILLED',
    'STOPPING_COMPLETED',
    'STOPPING_ERROR',
    'DELETING',
    'CANCELED',
    'COMPLETED',
    'ERROR',
    'DELETE_FAILED'
);
ALTER TABLE public.experiments ALTER COLUMN state TYPE experiment_state USING state::text::experiment_state;
DROP TYPE _experiment_state;

ALTER TYPE public.trial_state RENAME TO _trial_state;
CREATE TYPE public.trial_state AS ENUM (
    'ACTIVE',
    'PAUSED',
    'STOPPING_CANCELED',
    'STOPPING_KILLED',
    'STOPPING_COMPLETED',
    'STOPPING_ERROR',
    'CANCELED',
    'COMPLETED',
    'ERROR'
);
ALTER TABLE public.trials ALTER COLUMN state TYPE trial_state USING state::text::trial_state;
DROP TYPE _trial_state;

ALTER TABLE public.trials
    ADD COLUMN task_id text NULL,
    ADD COLUMN run_id integer NOT NULL DEFAULT 0,
    ADD COLUMN restarts integer NOT NULL DEFAULT 0;

ALTER TABLE public.raw_steps
    ADD COLUMN computed_records integer NULL;

ALTER TABLE public.raw_validations
    ADD COLUMN computed_records integer NULL;

CREATE TYPE public.task_type AS ENUM (
    'TRIAL',
    'NOTEBOOK',
    'SHELL',
    'COMMAND',
    'TENSORBOARD',
    'CHECKPOINT_GC'
);

CREATE TABLE public.tasks (
    task_id text NOT NULL UNIQUE,
    task_type task_type NOT NULL,
    start_time timestamp without time zone NOT NULL,
    end_time timestamp without time zone NULL
);

CREATE TABLE public.allocations (
    task_id text NOT NULL REFERENCES public.tasks(task_id),
    allocation_id text NOT NULL UNIQUE,
    resource_pool text NOT NULL,
    -- Could store all reservations in-line if needed down the road.
    start_time timestamp without time zone NOT NULL,
    end_time timestamp without time zone NULL
);
