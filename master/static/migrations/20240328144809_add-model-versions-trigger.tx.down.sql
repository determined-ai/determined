DROP TRIGGER IF EXISTS stream_model_version_trigger_seq ON model_versions
DROP TRIGGER IF EXISTS stream_model_version_trigger_iu ON model_versions;
DROP TRIGGER IF EXISTS stream_model_version_trigger_d ON model_versions;
DROP TRIGGER IF EXISTS stream_model_version_trigger_by_model ON models;
DROP TRIGGER IF EXISTS stream_model_version_trigger_by_model_iu ON models;

DROP FUNCTION IF EXISTS stream_model_version_seq_modify;
DROP FUNCTION IF EXISTS stream_model_version_notify;
DROP FUNCTION IF EXISTS stream_model_version_change;
DROP FUNCTION IF EXISTS stream_model_version_seq_modify_by_model;
DROP FUNCTION IF EXISTS stream_model_version_change_by_model;

DROP SEQUENCE IF EXISTS stream_model_version_seq;
ALTER TABLE model_versions DROP COLUMN IF EXISTS seq;
