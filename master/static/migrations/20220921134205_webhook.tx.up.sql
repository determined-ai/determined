CREATE TABLE webhooks (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
)

CREATE TYPE public.trigger_type as ENUM (
    'EXPERIMENT_STATE_CHANGE',
    'METRIC_THRESHOLD_EXCEEDED'
)

CREATE TABLE webhook_triggers (
    id SERIAL PRIMARY KEY,
    type public.trigger_type NOT NULL,
    trigger JSONB,
    webhook_id integer REFERENCES webhooks(id)
)

CREATE TABLE webhook_events (
    id SERIAL PRIMARY KEY,
    trigger_id integer REFERENCES webhook_triggers(id) NOT NULL,
    payload JSONB,    
    attempts integer,
)