/*
table jobs
tasks => jobs
experiments => jobs
NTbCS => jobs (if they did persist in DB)
*/

CREATE TYPE public.job_type AS ENUM (
    'EXPERIMENT',
    'NOTEBOOK',
    'SHELL',
    'COMMAND',
    'TENSORBOARD'
);

CREATE TABLE public.jobs (
    job_id text NOT NULL UNIQUE,
    job_type job_type NOT NULL
);


-- TODO existing tasks that had jobs? do we care
ALTER TABLE public.tasks
    ADD COLUMN job_id text NOT NULL REFERENCES public.jobs(job_id);

-- FIXME maybe we allow nulls for old experiments?
ALTER TABLE public.experiments
    ADD COLUMN job_id text NOT NULL REFERENCES public.jobs(job_id);
