CREATE FUNCTION webhook_events_update_trigger()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('webhook_events:updated', NEW.id::text);
  RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER webhook_events_update_trigger
AFTER UPDATE ON webhook_events
FOR EACH ROW EXECUTE PROCEDURE webhook_events_update_trigger();