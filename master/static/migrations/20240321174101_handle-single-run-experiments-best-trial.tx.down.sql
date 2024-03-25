DROP TRIGGER autoupdate_exp_best_trial_single_run_no_validation_trigger ON runs;
UPDATE experiments
SET best_trial_id = NULL 
WHERE 
    best_trial_id IS NOT NULL AND
    best_trial_id IN (
        SELECT id
        FROM runs r
        WHERE 
            r.experiment_id = experiments.id 
            AND 
            r.best_validation_id IS NULL
    );
