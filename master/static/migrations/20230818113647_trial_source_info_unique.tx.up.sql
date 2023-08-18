ALTER TABLE ONLY public.trial_source_infos
    ADD CONSTRAINT trial_source_info_trial_ckpt_key UNIQUE (trial_id, checkpoint_uuid);