ALTER TYPE public.experiment_state RENAME TO _experiment_state;
CREATE TYPE public.experiment_state AS ENUM (
    'ACTIVE',
    'CANCELED',
    'COMPLETED',
    'ERROR',
    'PAUSED',
    'STOPPING_CANCELED',
    'STOPPING_KILLED',
    'STOPPING_COMPLETED',
    'STOPPING_ERROR',
    'DELETING',
    'DELETE_FAILED'
);
ALTER TABLE public.experiments ALTER COLUMN state TYPE experiment_state USING state::text::experiment_state;
DROP TYPE _experiment_state;

ALTER TABLE public.trials
    ADD COLUMN run_id integer NOT NULL DEFAULT 0,
    ADD COLUMN restarts integer NOT NULL DEFAULT 0;

DROP TABLE public.trial_snapshots;

DROP TABLE public.runs;

DROP TYPE public.run_type;

CREATE TYPE public.job_type AS ENUM (
    'TRIAL',
    'NOTEBOOK',
    'SHELL',
    'COMMAND',
    'CHECKPOINT_GC'
);

CREATE TABLE public.tasks (
    id SERIAL,
    task_id text NOT NULL,
    start_time timestamp without time zone NULL,
    end_time timestamp without time zone NULL,
    job_type job_type NOT NULL,
    job_type_fk_id integer NOT NULL,
    CONSTRAINT tasks_job_type_job_type_fk_id_task_id_unique UNIQUE (job_type, job_type_fk_id, task_id)
);
