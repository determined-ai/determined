WITH inf_trial AS (
    SELECT
        ts.trial_id as trial_id,
        ts.checkpoint_uuid,
        ts.source_trial_id
    FROM trial_source_info ts
    LEFT JOIN public.trials t1 ON ts.trial_id = t1.id
    LEFT JOIN public.trials t2 ON ts.source_trial_id = t2.id
    WHERE ts.source_trial_id = 640
)
SELECT trial_id FROM inf_trial;
