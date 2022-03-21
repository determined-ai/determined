CREATE TABLE public.agent_stats (
    resource_pool text NOT NULL,
    agent_id text NOT NULL,
    slots smallint NOT NULL DEFAULT 1,
    start_time timestamptz NOT NULL,
    end_time timestamptz NULL
);

