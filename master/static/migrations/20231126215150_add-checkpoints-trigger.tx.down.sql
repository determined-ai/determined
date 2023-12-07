DROP TRIGGER IF EXISTS stream_checkpoint_trigger_d ON checkpoints_v2;
DROP TRIGGER IF EXISTS stream_checkpoint_trigger_iu ON checkpoints_v2;
DROP TRIGGER IF EXISTS stream_checkpoint_trigger_seq ON checkpoints_v2;
DROP TRIGGER IF EXISTS stream_checkpoint_workspace_change_trigger ON projects;
DROP TRIGGER IF EXISTS stream_checkpoint_project_change_trigger ON experiments;

DROP FUNCTION IF EXISTS stream_checkpoint_change;
DROP FUNCTION IF EXISTS stream_checkpoint_notify;
DROP FUNCTION IF EXISTS stream_checkpoint_seq_modify;
DROP FUNCTION IF EXISTS stream_checkpoint_workspace_change_notify;
DROP FUNCTION IF EXISTS stream_checkpoint_project_change_notify;

DROP SEQUENCE IF EXISTS stream_checkpoint_seq;
ALTER TABLE checkpoints_v2 DROP COLUMN IF EXISTS seq;
