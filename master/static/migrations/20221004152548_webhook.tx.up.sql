CREATE TYPE public.webhook_type AS ENUM (
    'DEFAULT',
    'SLACK'
);

CREATE TABLE webhooks (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    webhook_type public . WEBHOOK_TYPE NOT NULL
);

CREATE TYPE public.trigger_type AS ENUM (
    'EXPERIMENT_STATE_CHANGE',
    'METRIC_THRESHOLD_EXCEEDED'
);

CREATE TABLE webhook_triggers (
    id SERIAL PRIMARY KEY,
    trigger_type public . TRIGGER_TYPE NOT NULL,
    condition JSONB NOT NULL,
    webhook_id INTEGER NOT NULL REFERENCES webhooks (id) ON DELETE CASCADE
);

CREATE TABLE webhook_events (
    id SERIAL PRIMARY KEY,
    trigger_id INTEGER NOT NULL REFERENCES webhook_triggers (
        id
    ) ON DELETE CASCADE,
    attempts INTEGER DEFAULT 0,
    payload JSONB NOT NULL
);
