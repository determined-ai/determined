ALTER TABLE public.jobs
    ADD COLUMN owner_id integer REFERENCES public.users(id) NULL;

ALTER TABLE public.tasks
    ADD COLUMN job_id text REFERENCES public.jobs(job_id) NULL;
