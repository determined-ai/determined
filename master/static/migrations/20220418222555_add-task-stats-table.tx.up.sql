CREATE TYPE public.stats_type AS ENUM (
    'QUEUED',
    'IMAGEPULL'
);

CREATE TABLE public.task_stats (
    allocation_id text NOT NULL REFERENCES public.allocations(allocation_id),
    event_type public.stats_type NOT NULL,
    start_time timestamptz NOT NULL,
    end_time timestamptz NULL
);