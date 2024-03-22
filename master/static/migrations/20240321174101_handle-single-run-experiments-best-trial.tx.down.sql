DROP TRIGGER IF EXISTS autoupdate_exp_best_trial_metrics;
CREATE TRIGGER autoupdate_exp_best_trial_metrics
AFTER UPDATE OF best_validation_id ON runs
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_best_trial_metrics();

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
