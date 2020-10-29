ALTER TABLE public.trial_logs
    ADD COLUMN agent_id text NULL,
     -- In the case of k8s, this is a pod name.
    ADD COLUMN container_id text NULL,
    ADD COLUMN rank_id smallint NULL,
    -- For backward compatibility, add a new column for logs that have been parsed through Fluent
    -- Bit. New and old logs will be distinguished by whether the previous `message` column is
    -- present or this one.
    ADD COLUMN log bytea NULL,
    ADD COLUMN timestamp timestamp NULL,
    ADD COLUMN level text NULL,
    ADD COLUMN source text NULL,
    ADD COLUMN stdtype text NULL;
