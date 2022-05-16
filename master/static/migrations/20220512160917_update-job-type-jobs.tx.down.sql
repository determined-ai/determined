ALTER TABLE public.jobs
    ALTER COLUMN job_type TYPE VARCHAR(255);

DROP TYPE public.job_type;

CREATE TYPE public.job_type AS ENUM (
    'EXPERIMENT',
    'NOTEBOOK',
    'SHELL',
    'COMMAND',
    'TENSORBOARD'
);

ALTER TABLE public.jobs 
    ALTER COLUMN job_type TYPE public.job_type
    USING (job_type::job_type);
