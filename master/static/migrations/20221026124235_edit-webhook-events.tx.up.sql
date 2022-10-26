DROP TABLE IF EXISTS webhook_events;

CREATE TABLE IF NOT EXISTS webhook_events_queue (
  id SERIAL PRIMARY KEY,
  url text NOT NULL,
  payload bytea NOT NULL
);