ALTER TABLE public.trials
    ADD COLUMN run_start timestamp without time zone NULL,
    ADD COLUMN metadata jsonb NOT NULL DEFAULT '{}'::jsonb;

AlTER TABLE public.steps
    ADD COLUMN total_inputs integer NOT NULL DEFAULT 0;