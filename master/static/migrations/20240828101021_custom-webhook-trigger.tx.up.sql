ALTER TYPE trigger_type RENAME TO _trigger_type;

CREATE TYPE trigger_type AS ENUM (
  'EXPERIMENT_STATE_CHANGE',
  'METRIC_THRESHOLD_EXCEEDED',
  'TASK_LOG',
  'CUSTOM'
);

ALTER TABLE webhook_triggers ALTER COLUMN trigger_type
    SET DATA TYPE trigger_type USING (trigger_type::text::trigger_type);

DROP TYPE public._trigger_type;