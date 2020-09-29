ALTER TABLE public.trial_logs
    DROP COLUMN agent_id,
    DROP COLUMN container_id,
    DROP COLUMN timestamp,
    DROP COLUMN level,
    DROP COLUMN source,
    DROP COLUMN stream,
    DROP COLUMN rank_id;
