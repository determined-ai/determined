ALTER TYPE trigger_type RENAME TO _trigger_type;

CREATE TYPE trigger_type AS ENUM (
  'EXPERIMENT_STATE_CHANGE',
  'METRIC_THRESHOLD_EXCEEDED',
  'TASK_LOG'
);

ALTER TABLE webhook_triggers ALTER COLUMN trigger_type
    SET DATA TYPE trigger_type USING (trigger_type::text::trigger_type);

DROP TYPE public._trigger_type;

CREATE TABLE webhook_task_log_triggers (
    task_id text  NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    trigger_id integer NOT NULL REFERENCES webhook_triggers(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, trigger_id)
);
