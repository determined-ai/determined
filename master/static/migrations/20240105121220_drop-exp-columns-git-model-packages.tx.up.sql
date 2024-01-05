ALTER TABLE public.experiments
    DROP COLUMN git_remote,
    DROP COLUMN git_commit,
    DROP COLUMN git_committer,
    DROP COLUMN git_commit_date,
    DROP COLUMN model_packages;