CREATE TYPE public.webhook_type AS ENUM (
    'DEFAULT',
    'SLACK'
);

CREATE TABLE webhooks (
    id serial PRIMARY KEY,
    url text NOT NULL,
    webhook_type public.webhook_type NOT NULL
);

CREATE TYPE public.trigger_type AS ENUM (
    'EXPERIMENT_STATE_CHANGE',
    'METRIC_THRESHOLD_EXCEEDED'
);

CREATE TABLE webhook_triggers (
    id serial PRIMARY KEY,
    trigger_type public.trigger_type NOT NULL,
    condition jsonb NOT NULL,
    webhook_id integer NOT NULL REFERENCES webhooks (id) ON DELETE CASCADE
);

CREATE TABLE webhook_events (
    id serial PRIMARY KEY,
    trigger_id integer NOT NULL REFERENCES webhook_triggers (id) ON DELETE CASCADE,
    attempts integer DEFAULT 0,
    payload jsonb NOT NULL
);

