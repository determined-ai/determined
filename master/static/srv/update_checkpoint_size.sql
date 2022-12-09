WITH update_trial AS (
    UPDATE trials SET checkpoint_size = (
    SELECT coalesce(sum((size_tuple).value::text::bigint), 0)
    FROM (
    SELECT jsonb_each(c.resources) AS size_tuple
            FROM checkpoints_view c
            WHERE state != 'DELETED'
            AND trial_id = (SELECT trial_id from checkpoints_view where uuid = $1 LIMIT 1)
            AND c.resources != 'null'::jsonb) r)
    WHERE trials.id = (SELECT trial_id from checkpoints_view where uuid = $1 LIMIT 1)
    RETURNING id),  
update_experiment AS (
UPDATE experiments SET (checkpoint_size, checkpoint_count) =  (
    SELECT coalesce(sum((size_tuple).value::text::bigint), 0), count(distinct(uuid))
    FROM (
    SELECT jsonb_each(c.resources) AS size_tuple, uuid
            FROM checkpoints_view c
            WHERE state != 'DELETED'
            AND experiment_id = (SELECT experiment_id from checkpoints_view where uuid = $1 LIMIT 1)
            AND c.resources != 'null'::jsonb) r)
    WHERE experiments.id = (SELECT experiment_id from checkpoints_view where uuid = $1 LIMIT 1)
RETURNING id
)

SELECT id from update_experiment
