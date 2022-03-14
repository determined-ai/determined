CREATE TABLE public.raw_agent (
    resource_pool text NOT NULL,
    agent_id text NULL,
    slots smallint NOT NULL DEFAULT 1,
    start_time timestamp without time zone NOT NULL,
    end_time timestamp without time zone NULL
);
