CREATE TYPE public.trial_source_info_type AS ENUM (
    'INFERENCE',
    'FINE_TUNING'
);

-- Denotes a connection between a given trial and a checkpoint/source trial/model version
CREATE TABLE public.trial_source_infos (
    -- Inference/Fine Tuning trial
    trial_id int REFERENCES public.trials(id) ON DELETE CASCADE NOT NULL,
    -- Checkpoint in question. Lifted from referred source_trial_id/source_model_version_id
    checkpoint_uuid uuid REFERENCES public.checkpoints_v2(uuid) ON DELETE CASCADE NOT NULL,
    -- Original trial that created the checkpoint (may be null in some inference and fine tuning use cases)
    source_trial_id int REFERENCES public.trials(id) ON DELETE CASCADE NULL,
    -- Source Trial's `model_version` `id` field
    source_model_version_id int NULL,
    -- Source Trial's `model_version` `version` field
    source_model_version_version int NULL,
    -- Type of the `trial_source_info` (inference or fine tuning for now)
    trial_source_info_type trial_source_info_type NOT NULL,
    -- User defined description text
    description text NULL,
    -- User defined metadata
    metadata jsonb NULL,

    CONSTRAINT fk_model_versions FOREIGN KEY (source_model_version_id, source_model_version_version) REFERENCES public.model_versions (model_id, version),
    -- `public.model_version` defines its primary key as the combination of these
    -- two values. Make sure that either they are both present or both missing
    CONSTRAINT check_model_version_valid CHECK (
        (source_model_version_id IS NULL AND source_model_version_version IS NULL) OR
        (
            (source_model_version_id IS NOT NULL AND source_model_version_version IS NOT NULL) AND
            (source_model_version_id > 0 AND source_model_version_version > 0)
        )
    )
);

CREATE INDEX ix_trial_source_infos_trial_id ON public.trial_source_infos USING btree (trial_id);
CREATE INDEX ix_trial_source_infos_checkpoint_uuid ON public.trial_source_infos USING btree (checkpoint_uuid);
CREATE INDEX ix_trial_source_infos_source_trial_id ON public.trial_source_infos USING btree (source_trial_id);
CREATE INDEX ix_trial_source_infos_source_model_version ON public.trial_source_infos USING btree (source_model_version_id, source_model_version_version);
