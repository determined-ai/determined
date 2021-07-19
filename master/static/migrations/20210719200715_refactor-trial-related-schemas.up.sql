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
