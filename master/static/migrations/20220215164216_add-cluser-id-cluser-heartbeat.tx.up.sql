ALTER TABLE public.cluster_id 
    ADD COLUMN cluster_heartbeat timestamp not null 
        DEFAULT (DATE_TRUNC('millisecond', now() at time zone 'utc')::timestamp);
