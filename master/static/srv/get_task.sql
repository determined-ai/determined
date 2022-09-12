SELECT tasks.task_id, (SELECT coalesce(jsonb_agg(allo ORDER BY end_time DESC NULLS FIRST), '[]'::jsonb) FROM (
  SELECT allocation_id, task_id, 'STATE_' || CASE WHEN (task_type = 'TENSORBOARD' AND state = 'PENDING') THEN 
    CASE WHEN EXISTS (
      select 1 from raw_steps, raw_validations, trials, tasks where trials.task_id=tasks.task_id and (raw_steps.trial_id=trials.id or raw_validations.trial_id=trials.id) 
    ) THEN 'PENDING' ELSE 'WAITING' END 
   ELSE state::text END AS state , is_ready, start_time, end_time
  FROM allocations
  WHERE allocations.task_id = tasks.task_id
) allo) AS allocations
FROM tasks
WHERE tasks.task_id = $1;
