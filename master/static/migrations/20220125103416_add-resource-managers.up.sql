CREATE TYPE public.resource_manager_type AS ENUM ('AGENT', 'KUBERNETES');
CREATE TABLE public.resource_managers (
    pool_name text NOT NULL,
    -- use RM abbreviation?
    rm_type resource_manager_type NOT NULL,
    state jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (pool_name, rm_type)
);
