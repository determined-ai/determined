DROP TABLE webhook_events;

CREATE TABLE webhook_events (
  id SERIAL PRIMARY KEY,
  trigger_id integer NOT NULL REFERENCES webhook_triggers(id) ON DELETE CASCADE ,
  payload bytea NOT NULL
);