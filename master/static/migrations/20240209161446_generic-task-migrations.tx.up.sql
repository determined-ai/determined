DO $$ 
BEGIN
    IF current_setting('server_version_num')::int > 120000 THEN
       ALTER TYPE task_type ADD VALUE 'GENERIC';
       ALTER TYPE job_type ADD VALUE 'GENERIC';
    ELSE
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
    END IF;
END $$;

CREATE TYPE public.task_state AS ENUM (
    'ACTIVE',
    'CANCELED',
    'COMPLETED',
    'ERROR',
    'PAUSED',
    'STOPPING_PAUSED',
    'STOPPING_CANCELED',
    'STOPPING_COMPLETED',
    'STOPPING_ERROR'
);

ALTER TABLE tasks
ADD config jsonb DEFAULT(NULL),
ADD forked_from text DEFAULT NULL,
ADD parent_id text REFERENCES tasks(task_id) ON DELETE CASCADE DEFAULT(NULL),
ADD task_state public.task_state,
ADD no_pause boolean DEFAULT NULL;

ALTER TABLE command_state
ADD generic_task_spec jsonb DEFAULT NULL;
