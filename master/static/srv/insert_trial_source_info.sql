INSERT INTO public.trial_source_info (
    trial_id,
    checkpoint_uuid,
    source_model_version_id,
    source_model_version_version,
    trial_source_info_type,
    description,
    metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING trial_id, checkpoint_uuid;
