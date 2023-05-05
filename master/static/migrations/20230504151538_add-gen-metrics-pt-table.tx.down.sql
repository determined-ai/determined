ALTER TABLE metrics DETACH PARTITION raw_steps;
ALTER TABLE metrics DETACH PARTITION raw_validations;
ALTER TABLE metrics DETACH PARTITION generic_metrics;

DROP TABLE metrics;
DROP TABLE generic_metrics;

DROP INDEX IF EXISTS steps_trial_id_total_batches_run_id_type_unique;
CREATE UNIQUE INDEX steps_trial_id_total_batches_run_id_unique ON raw_steps (
    trial_id, total_batches, trial_run_id
);
ALTER TABLE raw_steps DROP COLUMN IF EXISTS type;

DROP INDEX IF EXISTS validations_trial_id_total_batches_run_id_type_unique;
CREATE UNIQUE INDEX validations_trial_id_total_batches_run_id_unique ON raw_validations (
    trial_id, total_batches, trial_run_id
);
ALTER TABLE raw_validations DROP COLUMN IF EXISTS type;

DROP TYPE metric_partition_type;
