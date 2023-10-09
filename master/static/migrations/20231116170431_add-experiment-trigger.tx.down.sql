DROP TRIGGER IF EXISTS stream_experiment_trigger_d ON experiments;
DROP TRIGGER IF EXISTS stream_experiment_trigger_iu ON experiments;
DROP TRIGGER IF EXISTS stream_experiment_trigger_seq ON experiments;

DROP FUNCTION IF EXISTS stream_experiment_workspace_change_notify;
DROP FUNCTION IF EXISTS stream_experiment_change;
DROP FUNCTION IF EXISTS stream_experiment_notify;
DROP FUNCTION IF EXISTS stream_experiment_seq_modify;

DROP SEQUENCE IF EXISTS stream_experiment_seq;
ALTER TABLE experiments DROP COLUMN IF EXISTS seq;
