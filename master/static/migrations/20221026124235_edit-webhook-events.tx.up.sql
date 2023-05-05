DROP TABLE IF EXISTS webhook_events;

CREATE TABLE IF NOT EXISTS webhook_events_queue (
    id serial PRIMARY KEY,
    url text NOT NULL,
    payload bytea NOT NULL
);

