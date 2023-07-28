DROP TABLE public.trial_source_infos;
DROP TYPE public.trial_source_info_type;

DROP INDEX IF EXISTS ix_trial_source_infos_trial_id;
DROP INDEX IF EXISTS ix_trial_source_infos_checkpoint_uuid;
DROP INDEX IF EXISTS ix_trial_source_infos_model_version;
