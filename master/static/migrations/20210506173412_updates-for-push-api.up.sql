CREATE TABLE public.trial_runs (
    id integer,
    start_time timestamp with time zone NOT NULL DEFAULT now(),
    trial_id integer,
    CONSTRAINT fk_trial_runs_trials_trial_id FOREIGN KEY(trial_id) REFERENCES public.trials(id),
    CONSTRAINT trial_runs_id_trial_id_unique UNIQUE (trial_id, id)
);

ALTER TABLE public.trials
    ADD COLUMN metadata jsonb NOT NULL DEFAULT '{}'::jsonb;

AlTER TABLE public.steps
    ADD COLUMN total_records integer NOT NULL DEFAULT 0,
    ADD COLUMN total_epochs real NOT NULL DEFAULT 0,
    ADD COLUMN trial_run_id integer NOT NULL DEFAULT 0,
    ADD COLUMN archived boolean NOT NULL DEFAULT false,
    DROP CONSTRAINT steps_trial_total_batches_unique,
    ADD CONSTRAINT steps_trial_id_run_id_total_batches_unique UNIQUE (trial_id, trial_run_id, total_batches);

ALTER TABLE public.steps
    RENAME TO raw_steps;

-- XXX: Rename these files, check this view's performance.
CREATE VIEW steps AS
    SELECT * FROM raw_steps WHERE NOT archived;

ALTER TABLE public.validations
    ADD COLUMN total_records integer NOT NULL DEFAULT 0,
    ADD COLUMN total_epochs real NOT NULL DEFAULT 0,
    ADD COLUMN trial_run_id integer NOT NULL DEFAULT 0,
    ADD COLUMN archived boolean NOT NULL DEFAULT false,
    DROP CONSTRAINT validations_trial_total_batches_unique,
    ADD CONSTRAINT validations_trial_id_run_id_total_batches_unique UNIQUE (trial_id, trial_run_id, total_batches);

ALTER TABLE public.validations
    RENAME TO raw_validations;

CREATE VIEW validations AS
    SELECT * FROM raw_validations WHERE NOT archived;

ALTER TABLE public.checkpoints
    ADD COLUMN total_records integer NOT NULL DEFAULT 0,
    ADD COLUMN total_epochs real NOT NULL DEFAULT 0,
    ADD COLUMN trial_run_id integer NOT NULL DEFAULT 0,
    ADD COLUMN archived boolean NOT NULL DEFAULT false,
    DROP CONSTRAINT checkpoints_trial_total_batches_unique,
    ADD CONSTRAINT checkpoints_trial_id_run_id_total_batches_unique UNIQUE (trial_id, trial_run_id, total_batches);

ALTER TABLE public.checkpoints
    RENAME TO raw_checkpoints;

CREATE VIEW checkpoints AS
    SELECT * FROM raw_checkpoints WHERE NOT archived;