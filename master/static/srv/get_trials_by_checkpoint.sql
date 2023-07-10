-- WITH inf_trial AS (
--     SELECT
--         ts.trial_id as trial_id,
--         ts.checkpoint_uuid as checkpoint_uuid,
--         -- ts.source_trial_id as source_trial_id
--     FROM trial_source_info ts
--     -- LEFT JOIN public.trials t1 ON ts.trial_id = t1.id
--     -- LEFT JOIN public.trials t2 ON ts.source_trial_id = t2.id
--     WHERE ts.checkpoint_uuid = '17034885-d926-467e-a001-ea686356e3a7'
-- )
-- SELECT trial_id FROM inf_trial;

WITH trial_view AS (
    SELECT t.id as trial_id
    FROM trials AS t
    WHERE (t.id IN 
        (SELECT "trial_id" 
        FROM "trial_source_infos" 
        WHERE (checkpoint_uuid = 'ffe64b86-787c-4a3d-87fd-e398d3737266'))
    )
)
SELECT tv.trial_id, m.metrics--, m.total_batches, m.archived, m.id, m.trial_run_id, proto_time(m.end_time) AS end_time, m.custom_type
FROM trial_view AS tv
LEFT JOIN metrics AS m
ON tv.trial_id = m.trial_id;
-- WHERE (m.custom_type = 'inference') OR m.custom_type IS NULL;



-- SELECT t.id as trial_id, m.metrics, m.total_batches, m.archived, m.id, m.trial_run_id, proto_time(m.end_time) AS end_time, m.custom_type
-- FROM trial_view AS t
-- LEFT JOIN metrics AS m
-- ON m.trial_id = t.id 
-- -- WHERE (
-- --     (m.id IS NULL) 
-- --     OR ((m.partition_type = 'GENERIC') AND (m.archived = false) AND (m.custom_type = 'inference'))
-- -- )
-- -- AND (t.id IN 
-- -- -- WHERE (t.id IN 
-- --     (SELECT "trial_id" 
-- --     FROM "trial_source_infos" 
-- --     WHERE (checkpoint_uuid = 'ffe64b86-787c-4a3d-87fd-e398d3737266'))
-- -- )
-- ORDER BY t.id--, m.trial_run_id, m.total_batches LIMIT 1000
