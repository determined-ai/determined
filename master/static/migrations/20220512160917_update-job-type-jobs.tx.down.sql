ALTER TYPE public.job_type RENAME TO _job_type; 
CREATE TYPE public.job_type AS ENUM (
    'EXPERIMENT',
    'NOTEBOOK',
    'SHELL',
    'COMMAND',
    'TENSORBOARD'
);
DELETE FROM public.jobs WHERE job_type = 'CHECKPOINT_GC'
ALTER TABLE public.jobs ALTER COLUMN job_type TYPE public.job_type USING (job_type::text::job_type);
DROP TYPE _job_type;
