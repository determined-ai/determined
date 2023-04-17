DROP TABLE webhook_events_queue;

CREATE TABLE webhook_events (
    id SERIAL PRIMARY KEY,
    trigger_id INTEGER NOT NULL REFERENCES webhook_triggers (
        id
    ) ON DELETE CASCADE,
    attempts INTEGER DEFAULT 0,
    payload JSONB NOT NULL
);
