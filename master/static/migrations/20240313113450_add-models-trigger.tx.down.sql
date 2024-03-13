DROP TRIGGER IF EXISTS stream_model_trigger_seq ON models
DROP TRIGGER IF EXISTS stream_model_trigger_iu ON models;
DROP TRIGGER IF EXISTS stream_model_trigger_d ON models;

DROP FUNCTION IF EXISTS stream_model_seq_modify;
DROP FUNCTION IF EXISTS stream_model_notify;
DROP FUNCTION IF EXISTS stream_model_change;

DROP SEQUENCE IF EXISTS stream_model_seq;
ALTER TABLE models DROP COLUMN IF EXISTS seq;
