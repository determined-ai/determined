DROP INDEX trials_experiment_id_external_trial_id_unique;
ALTER TABLE experiments
    DROP COLUMN external_experiment_id;
ALTER TABLE trials
    DROP COLUMN external_trial_id;
