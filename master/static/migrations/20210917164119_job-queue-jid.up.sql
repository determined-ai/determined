CREATE TYPE public.task_type AS ENUM (
    'EXPERIMENT',
    'NOTEBOOK',
    'SHELL',
    'COMMAND',
    'TENSORBOARD',
);

CREATE TABLE public.jobs (
    job_id text NOT NULL UNIQUE,
    task_type task_type NOT NULL,
);


-- TODO existing tasks that had jobs? do we care
ALTER TABLE public.tasks
    job_id text REFERENCES public.jobs(job_id),
