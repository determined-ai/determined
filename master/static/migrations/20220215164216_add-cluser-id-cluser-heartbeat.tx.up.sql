ALTER TABLE public.cluster_id
    ADD COLUMN cluster_heartbeat timestamp NOT NULL DEFAULT (DATE_TRUNC('millisecond', now() at time zone 'utc')::timestamp);

