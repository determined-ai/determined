-- Replace step_id with total_batches in the checkpoints table.
ALTER TABLE checkpoints ADD COLUMN total_batches integer NOT NULL DEFAULT 0;

UPDATE checkpoints AS c
SET total_batches = COALESCE(
    (SELECT
           s.prior_batches_processed + s.num_batches AS total_batches
    FROM steps s
    WHERE c.step_id = s.id AND c.trial_id = s.trial_id), 0);

ALTER TABLE checkpoints
    ADD CONSTRAINT checkpoints_trial_total_batches_unique UNIQUE (trial_id, total_batches);

ALTER TABLE checkpoints DROP COLUMN step_id;

-- Replace step_id with total_batches in the validations table.
ALTER TABLE validations ADD COLUMN total_batches integer NOT NULL DEFAULT 0;

UPDATE validations AS v
SET total_batches = COALESCE(
    (SELECT
            s.prior_batches_processed + s.num_batches AS total_batches
    FROM steps s
    WHERE v.step_id = s.id AND v.trial_id = s.trial_id), 0);

ALTER TABLE validations
    ADD CONSTRAINT validations_trial_total_batches_unique UNIQUE (trial_id, total_batches);

ALTER TABLE validations DROP COLUMN step_id;

-- Add total_batches in the steps table.
ALTER TABLE steps ADD COLUMN total_batches integer NOT NULL DEFAULT 0;

UPDATE steps AS s
SET total_batches = COALESCE(s.prior_batches_processed + s.num_batches, 0);

ALTER TABLE steps
    ADD CONSTRAINT steps_trial_total_batches_unique UNIQUE (trial_id, total_batches);
