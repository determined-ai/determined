-- backfill existing single-trial experiments without best_trial_id
WITH single_run_experiments AS (
    SELECT 
        id
    FROM  
        experiments e
    WHERE
        e.config->'searcher'->>'name' = 'single'
),
br_no_validation AS (
    SELECT r.experiment_id, r.id, r.best_validation_id
    FROM 
        runs r 
        INNER JOIN single_run_experiments sre 
        ON r.experiment_id = sre.id
    ORDER BY searcher_metric_value_signed)
UPDATE 
    experiments 
SET 
    best_trial_id = brnv.id 
FROM 
    br_no_validation brnv
WHERE 
    experiments.id = brnv.experiment_id;
