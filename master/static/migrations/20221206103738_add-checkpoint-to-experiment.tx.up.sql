ALTER TABLE experiments
	ADD COLUMN checkpoint_size bigint;
ALTER TABLE experiments
	ADD COLUMN checkpoint_count int;
ALTER TABLE trials
	ADD COLUMN checkpoint_size bigint;
ALTER TABLE trials
	ADD COLUMN checkpoint_count int;

UPDATE trials set (checkpoint_size, checkpoint_count) = (size, count) FROM (
SELECT coalesce(sum((size_tuple).value::text::bigint), 0) AS size, count(distinct(uuid)) AS count, trial_id
FROM (
    SELECT jsonb_each(c.resources) AS size_tuple, trial_id, uuid
    FROM checkpoints_view c
    WHERE state != 'DELETED'
    AND c.resources != 'null'::jsonb ) r GROUP BY trial_id
) s WHERE 
trial_id = trials.id; 

UPDATE experiments set (checkpoint_size, checkpoint_count) = (size, count) FROM (
SELECT coalesce(sum(checkpoint_size), 0) AS size, coalesce(sum(checkpoint_count), 0) AS count, experiment_id
FROM trials GROUP BY experiment_id
) t WHERE experiments.id = experiment_id;