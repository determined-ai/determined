-- sequence for tracking event order
CREATE SEQUENCE IF NOT EXISTS stream_model_seq START 1;

ALTER TABLE models ADD COLUMN IF NOT EXISTS seq bigint DEFAULT 0;

-- trigger function to update sequence number on row modification
-- this should fire BEFORE so that it can modify NEW directly.
CREATE OR REPLACE FUNCTION stream_model_seq_modify() RETURNS TRIGGER AS $$
BEGIN
    NEW.seq = nextval('stream_model_seq');
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS stream_model_trigger_seq ON models;
CREATE TRIGGER stream_model_trigger_seq
    BEFORE INSERT OR UPDATE OF
    name, description, creation_time, last_updated_time, metadata, labels, user_id, archived, notes, workspace_id
                     ON models
                         FOR EACH ROW EXECUTE PROCEDURE stream_model_seq_modify();

-- helper function to create a model jsonb object for streaming
CREATE OR REPLACE FUNCTION stream_model_notify(
    before jsonb, after jsonb
) RETURNS integer AS $$
DECLARE
    output jsonb = NULL;
BEGIN
    IF before IS NOT NULL THEN
        output = jsonb_object_agg('before', before);
    END IF;
    IF after IS NOT NULL THEN
        IF output IS NULL THEN
            output = jsonb_object_agg('after', after);
        ELSE
            output = output || jsonb_object_agg('after', after);
    END IF;
END IF;
    PERFORM pg_notify('stream_model_chan', output::text);
return 0;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to NOTIFY the master of changes, for INSERT/UPDATE/DELETE.
CREATE OR REPLACE FUNCTION stream_model_change() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        PERFORM stream_model_notify(NULL, to_jsonb(NEW));
    ELSEIF (TG_OP = 'UPDATE') THEN
        PERFORM stream_model_notify(to_jsonb(OLD), to_jsonb(NEW));
    ELSEIF (TG_OP = 'DELETE') THEN
        PERFORM stream_model_notify(to_jsonb(OLD), NULL);
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;

-- INSERT and UPDATE should fire AFTER to guarantee to emit the final row value.
DROP TRIGGER IF EXISTS stream_model_trigger_iu ON models;
CREATE TRIGGER stream_model_trigger_iu
    AFTER INSERT OR UPDATE OF
    name, description, creation_time, last_updated_time, metadata, labels, user_id, archived, notes, workspace_id
                    ON models
                        FOR EACH ROW EXECUTE PROCEDURE stream_model_change();

DROP TRIGGER IF EXISTS stream_model_trigger_d ON models;
CREATE TRIGGER stream_model_trigger_d
    BEFORE DELETE ON models
    FOR EACH ROW EXECUTE PROCEDURE stream_model_change();
