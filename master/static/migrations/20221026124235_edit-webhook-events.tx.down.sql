DROP TABLE webhook_events_queue;

CREATE TABLE webhook_events (
  id SERIAL PRIMARY KEY,
  trigger_id integer NOT NULL REFERENCES webhook_triggers(id) ON DELETE CASCADE ,
  attempts integer DEFAULT 0,
  payload jsonb NOT NULL
);
