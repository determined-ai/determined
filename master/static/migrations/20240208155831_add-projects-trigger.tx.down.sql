DROP TRIGGER IF EXISTS stream_project_trigger_seq ON projects
DROP TRIGGER IF EXISTS stream_project_trigger_iu ON projects;
DROP TRIGGER IF EXISTS stream_project_trigger_d ON projects;

DROP FUNCTION IF EXISTS stream_project_seq_modify;
DROP FUNCTION IF EXISTS stream_project_notify;
DROP FUNCTION IF EXISTS stream_project_change;

DROP SEQUENCE IF EXISTS stream_project_seq;
ALTER TABLE projects DROP COLUMN IF EXISTS seq;
