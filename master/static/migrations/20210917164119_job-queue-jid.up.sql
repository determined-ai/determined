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


-- TODO existing entities that had jobs? do we care?  only nonterminal
-- FIXME maybe we allow nulls for old experiments?
-- 	RETURNING id: ERROR: insert or update on table "experiments" violates foreign key constraint "experiments_job_id_fkey" (SQLSTATE 23503)
ALTER TABLE public.experiments
    ADD COLUMN job_id text; --  FIXME NOT NULL REFERENCES public.jobs(job_id);

ALTER TABLE public.tasks
    -- 	RETURNING id: ERROR: insert or update on table "experiments" violates foreign key constraint "experiments_job_id_fkey" (SQLSTATE 23503)
    ADD COLUMN job_id text; -- NOT NULL REFERENCES public.jobs(job_id);
