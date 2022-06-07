DROP INDEX steps_trial_id_total_batches_run_id_unique;
DROP INDEX steps_archived;
ALTER TABLE raw_steps ADD CONSTRAINT steps_trial_id_run_id_total_batches_unique UNIQUE (trial_id, trial_run_id, total_batches);

