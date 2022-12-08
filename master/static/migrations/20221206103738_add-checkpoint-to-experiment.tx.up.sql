ALTER TABLE experiments
	ADD COLUMN checkpoint_size bigint;
ALTER TABLE experiments
	ADD COLUMN checkpoint_count int;
ALTER TABLE trials
	ADD COLUMN checkpoint_size bigint;

UPDATE experiments set (checkpoint_size, checkpoint_count) = (size, count) FROM (
SELECT coalesce(sum((size_tuple).value::text::bigint), 0) AS size, count(distinct(uuid)) AS count, experiment_id
FROM (
    SELECT jsonb_each(c.resources) AS size_tuple, experiment_id, uuid
    FROM checkpoints_view c
    WHERE state != 'DELETED'
    AND c.resources != 'null'::jsonb) r GROUP BY experiment_id
) s WHERE 
experiment_id = experiments.id; 

UPDATE trials set checkpoint_size = size FROM (
SELECT coalesce(sum((size_tuple).value::text::bigint), 0) AS size, trial_id
FROM (
    SELECT jsonb_each(c.resources) AS size_tuple, trial_id
    FROM checkpoints_view c
    WHERE state != 'DELETED'
    AND c.resources != 'null'::jsonb ) r GROUP BY trial_id
) s WHERE 
trial_id = trials.id; 