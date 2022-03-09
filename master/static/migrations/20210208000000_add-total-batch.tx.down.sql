-- Replace total_batches with step_id in the checkpoints table.
ALTER TABLE checkpoints ADD COLUMN step_id integer NOT NULL DEFAULT 0;

UPDATE checkpoints AS c
SET step_id = COALESCE(
    (SELECT
            s.id AS step_id
    FROM steps s
    WHERE c.total_batches = s.total_batches AND c.trial_id = s.trial_id), 0);

ALTER TABLE checkpoints
    ADD CONSTRAINT checkpoints_trial_step_unique UNIQUE (trial_id, step_id);

ALTER TABLE checkpoints DROP COLUMN total_batches;

-- Replace total_batches with step_id in the validations table.
ALTER TABLE validations ADD COLUMN step_id integer NOT NULL DEFAULT 0;

UPDATE validations AS v
SET step_id = COALESCE(
    (SELECT
            s.id AS step_id
    FROM steps s
    WHERE v.total_batches = s.total_batches AND v.trial_id = s.trial_id), 0);

ALTER TABLE validations
    ADD CONSTRAINT validations_trial_step_unique UNIQUE (trial_id, step_id);

ALTER TABLE validations DROP COLUMN total_batches;

-- Remove total_batches in the steps table.
ALTER TABLE steps DROP COLUMN total_batches;
