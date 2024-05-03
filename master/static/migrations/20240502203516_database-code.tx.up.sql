DO $$
BEGIN
   execute 'ALTER DATABASE "'||current_database()||'" SET SEARCH_PATH TO public,determined_code';
END
$$;

DROP VIEW public.proto_checkpoints_view;
DROP VIEW public.checkpoints_view;

DROP VIEW public.trials;

DROP VIEW steps;
DROP VIEW validations;
DROP VIEW validation_metrics;

DROP FUNCTION public.abort_checkpoint_delete CASCADE;
DROP FUNCTION public.autoupdate_exp_best_trial_metrics CASCADE;
DROP FUNCTION public.autoupdate_exp_best_trial_metrics_on_delete CASCADE;
DROP FUNCTION public.autoupdate_user_image_deleted CASCADE;
DROP FUNCTION public.autoupdate_user_image_modified CASCADE;
DROP FUNCTION public.get_raw_metric CASCADE;
DROP FUNCTION public.get_signed_metric CASCADE;
DROP FUNCTION public.page_info CASCADE;
DROP FUNCTION public.proto_time CASCADE;
DROP FUNCTION public.retention_timestamp CASCADE;
DROP FUNCTION public.set_modified_time CASCADE;
DROP FUNCTION public.stream_model_change CASCADE;
DROP FUNCTION public.stream_model_notify CASCADE;
DROP FUNCTION public.stream_model_seq_modify CASCADE;
DROP FUNCTION public.stream_model_version_change CASCADE;
DROP FUNCTION public.stream_model_version_change_by_model CASCADE;
DROP FUNCTION public.stream_model_version_notify CASCADE;
DROP FUNCTION public.stream_model_version_seq_modify CASCADE;
DROP FUNCTION public.stream_model_version_seq_modify_by_model CASCADE;
DROP FUNCTION public.stream_project_change CASCADE;
DROP FUNCTION public.stream_project_notify CASCADE;
DROP FUNCTION public.stream_project_seq_modify CASCADE;
DROP FUNCTION public.try_float8_cast CASCADE;

DROP AGGREGATE public.jsonb_collect(jsonb);
