CREATE TYPE public.activity_type AS ENUM (
    'GET'
);

CREATE TYPE public.entity_type AS ENUM (
    'Project'
);

CREATE TABLE activity (
    user_id integer REFERENCES users (id) ON DELETE CASCADE NULL,
    activity_time timestamp with time zone NOT NULL DEFAULT NOW(),
    activity_type public.activity_type NOT NULL,
    entity_type public.entity_type NOT NULL,
    entity_id integer NOT NULL,
    CONSTRAINT user_activity_unique UNIQUE (user_id, activity_type, entity_type, entity_id)
);

