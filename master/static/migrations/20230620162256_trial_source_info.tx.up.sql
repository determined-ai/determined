CREATE TYPE public.trial_source_info_type AS ENUM (
    'INFERENCE',
    'FINE_TUNING'
);

CREATE TABLE public.trial_source_info (
    -- Inference/Fine Tuning trial
    trial_id int REFERENCES public.trials(id) ON DELETE CASCADE NOT NULL,
    -- Checkpoint in question. Lifted from referred source_trial_id/source_model_version_id
    checkpoint_uuid uuid REFERENCES public.checkpoints_v2(uuid) ON DELETE CASCADE NOT NULL,
    -- Original trial that created the checkpoint (may be null in some inference and fine tuning use cases)
    source_trial_id int REFERENCES public.trials(id) ON DELETE CASCADE NULL,
    -- Source Trial's Model version
    -- source_model_version_id int REFERENCES public.model_versions(id) ON DELETE CASCADE NULL,
    -- Type of the `trial_source_info` (inference or fine tuning for now)
    trial_source_info_type trial_source_info_type NOT NULL,
    description text,
    metadata jsonb
);

CREATE INDEX ix_trial_source_info_trial_id ON public.trial_source_info USING btree (trial_id);
CREATE INDEX ix_trial_source_info_checkpoint_uuid ON public.trial_source_info USING btree (checkpoint_uuid);
CREATE INDEX ix_trial_source_info_source_trial_id ON public.trial_source_info USING btree (source_trial_id);
