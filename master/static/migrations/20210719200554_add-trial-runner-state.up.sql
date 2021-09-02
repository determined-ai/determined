ALTER TABLE public.trials
    ADD COLUMN runner_state text NOT NULL DEFAULT 'UNSPECIFIED';
