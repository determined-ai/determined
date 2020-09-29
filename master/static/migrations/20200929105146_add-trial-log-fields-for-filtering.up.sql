ALTER TABLE public.trial_logs
    ADD COLUMN agent_id text NULL,
    ADD COLUMN container_id text NULL, -- In the case of k8s, this is a pod name.
    ADD COLUMN rank_id smallint NULL,
    ADD COLUMN timestamp timestamp NULL,
    ADD COLUMN level text NULL,
    ADD COLUMN source text NULL,
    ADD COLUMN std_type smallint NULL;
