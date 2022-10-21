CREATE INDEX trial_id_total_batches_end_time_raw_steps ON public.raw_steps
    USING btree (trial_id, total_batches, end_time) WHERE archived = false;
