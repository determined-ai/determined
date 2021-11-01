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


INSERT INTO jobs (
    SELECT 'backfilled-' || id as job_id, 'EXPERIMENT' as job_type FROM experiments
);

ALTER TABLE public.experiments
    ADD COLUMN job_id text REFERENCES public.jobs(job_id);

UPDATE experiments
    SET job_id = 'backfilled-' || id
    WHERE job_id IS NULL;

ALTER TABLE public.experiments
    ALTER COLUMN job_id SET NOT NULL;
