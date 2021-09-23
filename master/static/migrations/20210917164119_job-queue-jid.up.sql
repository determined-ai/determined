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


BEGIN; -- FIXME costly? migrations cover this case.

-- FIXME or we limit these to non-terminal experiments only?
INSERT INTO jobs (
    SELECT id as job_id, 'EXPERIMENT' as job_type FROM experiments
);

/* SET CONSTRAINTS ALL DEFERRED; */

ALTER TABLE public.experiments
    ADD COLUMN job_id text REFERENCES public.jobs(job_id);

UPDATE experiments
SET job_id = id
WHERE job_id IS NULL;

ALTER TABLE public.experiments
    ALTER COLUMN job_id SET NOT NULL;

COMMIT;

-- TODO do we need to persist task association?
/* ALTER TABLE public.tasks */
/*     ADD COLUMN job_id text NOT NULL REFERENCES public.jobs(job_id); */
