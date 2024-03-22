CREATE OR REPLACE FUNCTION autoupdate_exp_best_trial_metrics() RETURNS trigger AS $$
BEGIN
    WITH bt AS (
        SELECT id, best_validation_id 
        FROM runs 
        WHERE experiment_id = NEW.experiment_id 
        ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE experiments SET best_trial_id = bt.id FROM bt
    WHERE experiments.id = NEW.experiment_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS autoupdate_exp_best_trial_metrics ON runs;
CREATE TRIGGER autoupdate_exp_best_trial_metrics
AFTER INSERT OR UPDATE OF best_validation_id ON runs
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_best_trial_metrics();

WITH single_run_experiments AS (
    SELECT experiment_id
    FROM runs r
    GROUP BY experiment_id
    HAVING COUNT(*) = 1),
br_no_validation AS (
    SELECT r.experiment_id, r.id, r.best_validation_id
    FROM 
        runs r 
        INNER JOIN single_run_experiments sre 
        ON r.experiment_id = sre.experiment_id
    WHERE best_validation_id IS NULL
    ORDER BY searcher_metric_value_signed)
UPDATE experiments SET best_trial_id = brnv.id FROM br_no_validation brnv
WHERE 
    experiments.best_trial_id IS NULL
    AND experiments.id = brnv.experiment_id;
