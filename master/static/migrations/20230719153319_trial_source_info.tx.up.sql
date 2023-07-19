CREATE TYPE public.trial_source_info_type AS ENUM (
    'TRIAL_SOURCE_INFO_TYPE_INFERENCE',
    'TRIAL_SOURCE_INFO_TYPE_FINE_TUNING'
);

-- Denotes a connection between a given trial and a checkpoint/source trial/model version
CREATE TABLE public.trial_source_infos (
    -- Inference/Fine Tuning trial
    trial_id int REFERENCES public.trials(id) ON DELETE CASCADE NOT NULL,
    -- Checkpoint in question. Lifted from referred source_trial_id/model_version_id
    -- Note: We are not using a proper foreign key because you cannot make a foreign key
    -- on a view, which we are using to support both checkpoint v1 and v2.
    checkpoint_uuid uuid NOT NULL, -- REFERENCES public.checkpoints_v2(uuid) ON DELETE CASCADE NOT NULL,
    -- ID of the `model_version` this trial is linked to
    model_version_id int NULL,
    -- Version of the `model_version` this trial is linked to
    model_version_version int NULL,
    -- Type of the `trial_source_info` (inference or fine tuning for now)
    trial_source_info_type trial_source_info_type NOT NULL,

    CONSTRAINT fk_model_versions FOREIGN KEY (model_version_id, model_version_version) REFERENCES public.model_versions (model_id, version),
    -- `public.model_version` defines its primary key as the combination of these
    -- two values. Make sure that either they are both present or both missing
    CONSTRAINT check_model_version_valid CHECK (
        (model_version_id IS NULL AND model_version_version IS NULL) OR
        (
            (model_version_id IS NOT NULL AND model_version_version IS NOT NULL) AND
            (model_version_id > 0 AND model_version_version > 0)
        )
    )
);

CREATE INDEX ix_trial_source_infos_trial_id ON public.trial_source_infos USING btree (trial_id);
CREATE INDEX ix_trial_source_infos_checkpoint_uuid ON public.trial_source_infos USING btree (checkpoint_uuid);
CREATE INDEX ix_trial_source_infos_model_version ON public.trial_source_infos USING btree (model_version_id, model_version_version);
