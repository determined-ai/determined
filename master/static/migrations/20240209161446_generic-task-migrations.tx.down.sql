DELETE FROM task_stats WHERE allocation_id IN
(SELECT allocations.allocation_id
 FROM allocations JOIN tasks ON allocations.task_id=tasks.task_id
 WHERE tasks.task_type='GENERIC');
DELETE FROM allocations WHERE task_id IN (SELECT tasks.task_id
 FROM allocations JOIN tasks ON allocations.task_id=tasks.task_id
 WHERE tasks.task_type='GENERIC');
DELETE FROM tasks WHERE task_type = 'GENERIC';

ALTER TYPE task_type RENAME TO _task_type;

CREATE TYPE task_type AS ENUM (
  'TRIAL',
  'NOTEBOOK',
  'SHELL',
  'COMMAND',
  'TENSORBOARD',
  'CHECKPOINT_GC'
);

ALTER TABLE tasks ALTER COLUMN task_type
    SET DATA TYPE task_type USING (task_type::text::task_type);

DROP TYPE public._task_type;

DELETE FROM jobs WHERE job_type = 'GENERIC';

ALTER TYPE job_type RENAME TO _job_type;

CREATE TYPE job_type AS ENUM (
  'EXPERIMENT',
  'NOTEBOOK',
  'SHELL',
  'COMMAND',
  'TENSORBOARD',
  'CHECKPOINT_GC'
);

ALTER TABLE jobs ALTER COLUMN job_type
    SET DATA TYPE job_type USING (job_type::text::job_type);

DROP TYPE public._job_type;

ALTER TABLE tasks
DROP config,
DROP forked_from,
DROP parent_id,
DROP task_state,
DROP no_pause;

DROP TYPE public.task_state;

ALTER TABLE command_state
DROP generic_task_spec;
