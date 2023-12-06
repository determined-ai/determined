DROP TRIGGER IF EXISTS stream_metric_project_change_trigger ON experiments;
DROP TRIGGER IF EXISTS stream_metric_workspace_change_trigger ON projects;
DROP TRIGGER IF EXISTS stream_metric_generic_metrics_trigger_i ON generic_metrics;
DROP TRIGGER IF EXISTS stream_metric_raw_validations_trigger_i ON raw_validations;
DROP TRIGGER IF EXISTS stream_metric_raw_steps_trigger_i ON raw_steps;
DROP TRIGGER IF EXISTS stream_metric_generic_metrics_trigger_seq ON generic_metrics;
DROP TRIGGER IF EXISTS stream_metric_raw_validations_trigger_seq ON raw_validations;
DROP TRIGGER IF EXISTS stream_metric_raw_steps_trigger_seq ON raw_steps;

DROP FUNCTION IF EXISTS stream_metric_change;
DROP FUNCTION IF EXISTS stream_metric_notify;
DROP FUNCTION IF EXISTS stream_metric_seq_modify;

DROP SEQUENCE IF EXISTS stream_metric_seq;
ALTER TABLE metrics DROP COLUMN IF EXISTS seq;
