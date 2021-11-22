ALTER TABLE public.jobs
    DROP COLUMN owner_id;

ALTER TABLE public.tasks
    DROP COLUMN job_id;
