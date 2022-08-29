UPDATE raw_steps update_steps
SET archived = TRUE
FROM (SELECT DISTINCT trial_id, total_batches, trial_run_id FROM raw_steps outer_steps
    WHERE archived IS FALSE AND 
        trial_run_id < (SELECT MAX(inner_steps.trial_run_id) FROM raw_steps inner_steps WHERE 
        outer_steps.trial_id=inner_steps.trial_id AND
        outer_steps.total_batches=inner_steps.total_batches)) AS subquery_res
WHERE subquery_res.trial_id=update_steps.trial_id AND
    subquery_res.total_batches=update_steps.total_batches AND
    subquery_res.trial_run_id=update_steps.trial_run_id;
