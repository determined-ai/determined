CREATE OR REPLACE FUNCTION autoupdate_exp_best_trial_single_run_no_validation() RETURNS trigger AS $$
BEGIN 
    WITH rc AS (
        SELECT COUNT(*) AS count
        FROM runs
        WHERE experiment_id = NEW.id),
    br AS (
        SELECT id, best_validation_id 
        FROM runs 
        WHERE 
            runs.experiment_id = NEW.id 
            AND
            best_validation_id IS NULL
        ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE 
        experiments 
    SET 
        best_trial_id = br.id
    FROM br, rc
    WHERE
        experiments.id = NEW.id
        AND
        rc.count = 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS autoupdate_exp_best_trial_single_run_no_validation_trigger ON experiments;
CREATE TRIGGER autoupdate_exp_best_trial_single_run_no_validation_trigger 
AFTER UPDATE OF state ON experiments
FOR EACH ROW 
    WHEN (NEW.state = 'COMPLETED' AND NEW.best_trial_id IS NULL)
    EXECUTE PROCEDURE autoupdate_exp_best_trial_single_run_no_validation();

-- backfill existing single-trial experiments without best_trial_id
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
