ALTER TABLE experiments
    ADD COLUMN external_experiment_id TEXT UNIQUE NULL;
ALTER TABLE trials
    ADD COLUMN external_trial_id TEXT NULL;
CREATE UNIQUE INDEX trials_experiment_id_external_trial_id_unique ON trials(experiment_id, external_trial_id);
