DROP TABLE webhook_events;

CREATE TABLE webhook_events (
  id SERIAL PRIMARY KEY,
  payload bytea NOT NULL
);
