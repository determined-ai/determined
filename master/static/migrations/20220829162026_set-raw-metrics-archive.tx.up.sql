UPDATE raw_steps update_steps
SET archived = TRUE
FROM ( -- Get max trial_run_id for each (trial_id, total_batches) pair
    SELECT trial_id, total_batches, max(trial_run_id) AS trial_run_id FROM raw_steps
    WHERE (total_batches, trial_id) IN (
        -- Only consider trials and steps with a dupe UUID in the checkpoints_view
        SELECT steps_completed, trial_id FROM (
            SELECT steps_completed, trial_id, ROW_NUMBER() 
                OVER(PARTITION BY uuid ORDER BY id asc) AS Row FROM checkpoints_view
        ) dups WHERE dups.Row > 1
    )
    GROUP BY (trial_id, total_batches)
) AS subquery_res
WHERE subquery_res.trial_id=update_steps.trial_id AND
    subquery_res.total_batches=update_steps.total_batches AND
    subquery_res.trial_run_id=update_steps.trial_run_id;
