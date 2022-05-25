WITH w AS (
  SELECT id
  FROM workspaces
  WHERE id = $1
  AND NOT immutable
  AND (user_id = $2 OR $3 IS TRUE)
),
pins AS (
  DELETE FROM workspace_pins
  WHERE workspace_id IN (SELECT id FROM w)
),
proj AS (
  SELECT id FROM projects
  WHERE workspace_id IN (SELECT id FROM w)
),
exper AS (
  SELECT id FROM experiments
  WHERE project_id IN (SELECT id FROM proj)
),
o_trials AS (
  SELECT id, task_id FROM trials
  WHERE experiment_id IN (SELECT id FROM exper)
),
del_steps AS (
  DELETE FROM raw_steps
  WHERE trial_id IN (SELECT id FROM o_trials)
),
del_trials AS (
  DELETE FROM trials
  WHERE experiment_id IN (SELECT id FROM exper)
),
o_allocations AS (
  SELECT allocation_id FROM allocations
  WHERE task_id IN (SELECT task_id FROM o_trials)
),
del_task_stats AS (
  DELETE FROM task_stats
  WHERE allocation_id IN (SELECT allocation_id FROM o_allocations)
),
del_allocations AS (
  DELETE FROM allocations
  WHERE allocation_id IN (SELECT allocation_id FROM o_allocations)
),
del_tasks AS (
  DELETE FROM tasks
  WHERE task_id IN (SELECT task_id FROM o_trials)
),
del_exper AS (
  DELETE FROM experiments
  WHERE id IN (SELECT id FROM exper)
),
del_proj AS (
  DELETE FROM projects
  WHERE id IN (SELECT id FROM proj)
)
DELETE FROM workspaces
WHERE id IN (SELECT id FROM w)
RETURNING id;
