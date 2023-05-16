ALTER TABLE public.cluster_id 
ADD COLUMN cluster_heartbeat timestamp NOT NULL 
DEFAULT (DATE_TRUNC('millisecond', NOW() AT TIME ZONE 'utc')::timestamp);
