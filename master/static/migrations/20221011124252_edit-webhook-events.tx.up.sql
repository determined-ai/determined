DROP TABLE webhook_events;

CREATE TABLE webhook_events_queue (
  id SERIAL PRIMARY KEY,
  url text NOT NULL,
  payload bytea NOT NULL
);