CREATE TABLE public.raw_instance (
    resource_pool text NOT NULL,
    instance_id text NULL,
    slots smallint NOT NULL DEFAULT 1,
    start_time timestamp without time zone NOT NULL,
    end_time timestamp without time zone NULL
);
