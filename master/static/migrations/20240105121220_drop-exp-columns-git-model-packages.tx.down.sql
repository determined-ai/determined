ALTER TABLE public.experiments
    ADD COLUMN git_remote character,
    ADD COLUMN git_commit character,
    ADD COLUMN git_commit character,
    ADD COLUMN git_commit_date timestamp without time zone,
    ADD COLUMN model_packages bytea;
