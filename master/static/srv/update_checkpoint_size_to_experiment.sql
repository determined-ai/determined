UPDATE experiments set checkpoint_size = (
    SELECT coalesce(sum((size_tuple).value::text::bigint), 0)
    from (
    SELECT jsonb_each(c.resources) AS size_tuple
            FROM checkpoints_view c
            WHERE state != 'DELETED'
            AND experiment_id = (SELECT experiment_id from checkpoints_view where task_id = $1 LIMIT 1)
            AND c.resources != 'null'::jsonb) r)
    WHERE experiments.id = (SELECT experiment_id from checkpoints_view where task_id = $1 LIMIT 1)
RETURNING id