DROP TRIGGER IF EXISTS stream_trial_project_change_trigger ON experiments;
DROP TRIGGER IF EXISTS stream_trial_workspace_change_trigger ON projects;
DROP TRIGGER IF EXISTS stream_trial_trigger_d ON trials;
DROP TRIGGER IF EXISTS stream_trial_trigger_iu ON trials;
DROP TRIGGER IF EXISTS stream_trial_trigger_seq ON trials;

DROP FUNCTION IF EXISTS stream_trial_project_change_notify;
DROP FUNCTION IF EXISTS stream_trial_workspace_change_notify;
DROP FUNCTION IF EXISTS stream_trial_change;
DROP FUNCTION IF EXISTS stream_trial_notify;
DROP FUNCTION IF EXISTS stream_trial_seq_modify;

DROP SEQUENCE IF EXISTS stream_trial_seq;
ALTER TABLE trials DROP COLUMN IF EXISTS seq;
