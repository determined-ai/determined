DROP VIEW trials;

ALTER TABLE runs RENAME COLUMN restart_id TO run_id;
ALTER TABLE runs RENAME COLUMN external_run_id TO external_trial_id;
ALTER TABLE runs RENAME TO trials;
