-- all we really need
CREATE INDEX steps_archived ON raw_steps(archived);


ALTER TABLE raw_steps DROP CONSTRAINT steps_trial_id_run_id_total_batches_unique;

-- not needed, but presumably better? no performance sensitive queries
-- check based on trian_run_id, so better to put it after?
CREATE UNIQUE INDEX steps_trial_id_total_batches_run_id_unique ON raw_steps(trial_id, total_batches, trial_run_id);


-- not needed for existing queries to be performant, but maybe worth it?
-- CREATE UNIQUE INDEX steps_end_time ON raw_steps(end_time);
