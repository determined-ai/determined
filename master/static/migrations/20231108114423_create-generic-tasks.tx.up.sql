ALTER TYPE task_type RENAME TO _task_type;

CREATE TYPE task_type AS ENUM (
  'TRIAL',
  'NOTEBOOK',
  'SHELL',
  'COMMAND',
  'TENSORBOARD',
  'CHECKPOINT_GC',
  'GENERIC'
);

ALTER TABLE tasks ALTER COLUMN task_type
    SET DATA TYPE task_type USING (task_type::text::task_type);

DROP TYPE public._task_type;

ALTER TYPE job_type RENAME TO _job_type;

CREATE TYPE job_type AS ENUM (
  'EXPERIMENT',
  'NOTEBOOK',
  'SHELL',
  'COMMAND',
  'TENSORBOARD',
  'CHECKPOINT_GC',
  'GENERIC'
);

ALTER TABLE jobs ALTER COLUMN job_type
    SET DATA TYPE job_type USING (job_type::text::job_type);

DROP TYPE public._job_type;