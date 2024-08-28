DROP VIEW IF EXISTS proto_checkpoints_view;
DROP VIEW IF EXISTS checkpoints_view;

DROP VIEW IF EXISTS trials;

DROP VIEW IF EXISTS steps;
DROP VIEW IF EXISTS validations;
DROP VIEW IF EXISTS validation_metrics;

DROP FUNCTION IF EXISTS abort_checkpoint_delete CASCADE;
DROP FUNCTION IF EXISTS autoupdate_exp_best_trial_metrics CASCADE;
DROP FUNCTION IF EXISTS autoupdate_exp_best_trial_metrics_on_delete CASCADE;
DROP FUNCTION IF EXISTS autoupdate_user_image_deleted CASCADE;
DROP FUNCTION IF EXISTS autoupdate_user_image_modified CASCADE;
DROP FUNCTION IF EXISTS get_raw_metric CASCADE;
DROP FUNCTION IF EXISTS get_signed_metric CASCADE;
DROP FUNCTION IF EXISTS page_info CASCADE;
DROP FUNCTION IF EXISTS proto_time CASCADE;
DROP FUNCTION IF EXISTS retention_timestamp CASCADE;
DROP FUNCTION IF EXISTS set_modified_time CASCADE;
DROP FUNCTION IF EXISTS stream_model_change CASCADE;
DROP FUNCTION IF EXISTS stream_model_notify CASCADE;
DROP FUNCTION IF EXISTS stream_model_seq_modify CASCADE;
DROP FUNCTION IF EXISTS stream_model_version_change CASCADE;
DROP FUNCTION IF EXISTS stream_model_version_change_by_model CASCADE;
DROP FUNCTION IF EXISTS stream_model_version_notify CASCADE;
DROP FUNCTION IF EXISTS stream_model_version_seq_modify CASCADE;
DROP FUNCTION IF EXISTS stream_model_version_seq_modify_by_model CASCADE;
DROP FUNCTION IF EXISTS stream_project_change CASCADE;
DROP FUNCTION IF EXISTS stream_project_notify CASCADE;
DROP FUNCTION IF EXISTS stream_project_seq_modify CASCADE;
DROP FUNCTION IF EXISTS try_float8_cast CASCADE;

DROP AGGREGATE IF EXISTS jsonb_collect(jsonb);
DROP FUNCTION IF EXISTS prevent_auto_created_namespace_change CASCADE;
