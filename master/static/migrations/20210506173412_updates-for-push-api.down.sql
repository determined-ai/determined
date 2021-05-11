ALTER TABLE public.trials
    DROP COLUMN metadata;

DROP TABLE public.trial_runs;

AlTER TABLE public.steps
    DROP COLUMN total_inputs,
    DROP COLUMN trial_run_id,
    ADD CONSTRAINT steps_trial_total_batches_unique UNIQUE (trial_id, total_batches);

ALTER TABLE public.raw_steps RENAME TO steps;
DROP VIEW steps;

ALTER TABLE public.validations
    DROP COLUMN total_inputs,
    DROP COLUMN trial_run_id,
    ADD CONSTRAINT validations_trial_total_batches_unique UNIQUE (trial_id, total_batches);

ALTER TABLE public.raw_validations RENAME TO validations;
DROP VIEW steps;

ALTER TABLE public.checkpoints
    DROP COLUMN total_inputs,
    DROP COLUMN trial_run_id,
    ADD CONSTRAINT checkpoints_trial_total_batches_unique UNIQUE (trial_id, total_batches);

ALTER TABLE public.raw_checkpoints RENAME TO checkpoints;
DROP VIEW checkpoints;