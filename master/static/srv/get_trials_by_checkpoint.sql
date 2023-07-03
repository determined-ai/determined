WITH inf_trial AS (
    SELECT
        ts.trial_id as trial_id,
        ts.checkpoint_uuid as checkpoint_uuid,
        ts.source_trial_id as source_trial_id
    FROM trial_source_info ts
    -- LEFT JOIN public.trials t1 ON ts.trial_id = t1.id
    -- LEFT JOIN public.trials t2 ON ts.source_trial_id = t2.id
    WHERE ts.checkpoint_uuid = '17034885-d926-467e-a001-ea686356e3a7'
)
SELECT trial_id FROM inf_trial;