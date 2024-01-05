ALTER TABLE public.experiments
    ADD COLUMN git_remote character NULL,
    ADD COLUMN git_commit character NULL,
    ADD COLUMN git_commit character NULL,
    ADD COLUMN git_commit_date timestamp without time zone NULL,
    ADD COLUMN model_packages bytea NULL;