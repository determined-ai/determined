DROP TRIGGER IF EXISTS stream_model_version_trigger_seq ON model_versions
DROP TRIGGER IF EXISTS stream_model_version_trigger_iu ON model_versions;
DROP TRIGGER IF EXISTS stream_model_version_trigger_d ON model_versions;

DROP FUNCTION IF EXISTS stream_model_version_seq_modify;
DROP FUNCTION IF EXISTS stream_model_version_notify;
DROP FUNCTION IF EXISTS stream_model_version_change;

DROP SEQUENCE IF EXISTS stream_model_version_seq;
ALTER TABLE model_versions DROP COLUMN IF EXISTS seq;
