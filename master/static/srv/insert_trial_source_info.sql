INSERT INTO public.trial_source_info (trial_id, checkpoint_uuid)
VALUES ($1, $2, $3)
RETURNING (trial_id, checkpoint_uuid, trial_source_info_type)

-- TODO: Remove this
-- INSERT INTO public.trial_source_info (trial_id, checkpoint_uuid, trial_source_info_type)
-- VALUES (359, '7ba04ce0-73f1-463e-b348-a851a38b15e2', 'INFERENCE')
-- RETURNING (trial_id, checkpoint_uuid, trial_source_info_type)
