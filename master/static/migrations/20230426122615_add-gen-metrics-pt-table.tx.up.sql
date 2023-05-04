CREATE TYPE metric_type AS ENUM ('validation', 'training', 'generic');

ALTER TABLE raw_validations ADD COLUMN type metric_type NOT NULL DEFAULT 'validation';
ALTER INDEX validations_trial_id_total_batches_run_id_unique RENAME TO validations_trial_id_total_batches_run_id_unique_old;
CREATE UNIQUE INDEX validations_trial_id_total_batches_run_id_type_unique ON raw_validations (
    trial_id, total_batches, trial_run_id, type -- CHECK: safe to use `type` as the name?
);
DROP INDEX validations_trial_id_total_batches_run_id_unique_old;

ALTER TABLE raw_steps ADD COLUMN type metric_type NOT NULL DEFAULT 'training';
ALTER INDEX steps_trial_id_total_batches_run_id_unique RENAME TO steps_trial_id_total_batches_run_id_unique_old;
CREATE UNIQUE INDEX steps_trial_id_total_batches_run_id_type_unique ON raw_steps (
    trial_id, total_batches, trial_run_id, type
);
DROP INDEX steps_trial_id_total_batches_run_id_unique_old;

CREATE TABLE generic_metrics (LIKE raw_steps INCLUDING ALL);
ALTER TABLE generic_metrics ALTER COLUMN type SET DEFAULT 'generic';
CREATE UNIQUE INDEX generic_trial_id_total_batches_run_id_type_unique ON generic_metrics (
    trial_id, total_batches, trial_run_id, type
);

CREATE TABLE metrics (LIKE generic_metrics INCLUDING ALL)
PARTITION BY LIST (type);

ALTER TABLE metrics ATTACH PARTITION generic_metrics FOR VALUES IN ('generic');
ALTER TABLE metrics ATTACH PARTITION raw_validations FOR VALUES IN (
    'validation'
);
ALTER TABLE metrics ATTACH PARTITION raw_steps FOR VALUES IN ('training');
