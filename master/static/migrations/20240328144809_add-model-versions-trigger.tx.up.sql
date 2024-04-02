-- sequence for tracking event order
CREATE SEQUENCE IF NOT EXISTS stream_model_version_seq START 1;

ALTER TABLE model_versions ADD COLUMN IF NOT EXISTS seq bigint DEFAULT 0;

-- trigger function to update sequence number on row modification
-- this should fire BEFORE so that it can modify NEW directly.
CREATE OR REPLACE FUNCTION stream_model_version_seq_modify() RETURNS TRIGGER AS $$
BEGIN
    NEW.seq = nextval('stream_model_version_seq');
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION stream_model_version_seq_modify_by_model() RETURNS TRIGGER AS $$
BEGIN
    UPDATE model_versions SET seq = nextval('stream_model_version_seq') WHERE model_id = NEW.id;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS stream_model_version_trigger_seq ON model_versions;
CREATE TRIGGER stream_model_version_trigger_seq
    BEFORE INSERT OR UPDATE OF
    name, version, checkpoint_uuid, last_updated_time, metadata, labels, user_id, model_id, notes, comment 
                     ON model_versions
                         FOR EACH ROW EXECUTE PROCEDURE stream_model_version_seq_modify();

DROP TRIGGER IF EXISTS stream_model_version_trigger_by_model ON models;
CREATE TRIGGER stream_model_version_trigger_by_model
BEFORE UPDATE OF workspace_id ON models FOR EACH ROW EXECUTE PROCEDURE stream_model_version_seq_modify_by_model();


-- helper function to create a model version jsonb object for streaming
CREATE OR REPLACE FUNCTION stream_model_version_notify(
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
    PERFORM pg_notify('stream_model_version_chan', output::text);
return 0;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to NOTIFY the master of changes, for INSERT/UPDATE/DELETE.
CREATE OR REPLACE FUNCTION stream_model_version_change() RETURNS TRIGGER AS $$
DECLARE
    n jsonb = NULL;
    o jsonb = NULL;
BEGIN 
    IF (TG_OP = 'INSERT') THEN
        n = jsonb_set(to_jsonb(NEW), '{workspace_id}', to_jsonb((SELECT workspace_id FROM models WHERE id = NEW.model_id)::text));
        PERFORM stream_model_version_notify(NULL, n);
    ELSEIF (TG_OP = 'UPDATE') THEN
        n = jsonb_set(to_jsonb(NEW), '{workspace_id}', to_jsonb((SELECT workspace_id FROM models WHERE id = NEW.model_id)::text));
        o = jsonb_set(to_jsonb(OLD), '{workspace_id}', to_jsonb((SELECT workspace_id FROM models WHERE id = OLD.model_id)::text));
        PERFORM stream_model_version_notify(o, n);
    ELSEIF (TG_OP = 'DELETE') THEN
        o = jsonb_set(to_jsonb(OLD), '{workspace_id}', to_jsonb((SELECT workspace_id FROM models WHERE id = OLD.model_id)::text));
        PERFORM stream_model_version_notify(o, NULL);
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION stream_model_version_change_by_model() RETURNS TRIGGER AS $$
DECLARE
    f record;
BEGIN
    FOR f in (SELECT * FROM model_versions WHERE model_id = NEW.id)
    LOOP
        PERFORM stream_model_version_notify(NULL, jsonb_set(to_jsonb(f), '{workspace_id}', to_jsonb(NEW.workspace_id::text)));
    END LOOP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- INSERT and UPDATE should fire AFTER to guarantee to emit the final row value.
DROP TRIGGER IF EXISTS stream_model_version_trigger_iu ON model_versions;
CREATE TRIGGER stream_model_version_trigger_iu
    AFTER INSERT OR UPDATE OF
    name, version, checkpoint_uuid, last_updated_time, metadata, labels, user_id, model_id, notes, comment
                    ON model_versions
                        FOR EACH ROW EXECUTE PROCEDURE stream_model_version_change();

DROP TRIGGER IF EXISTS stream_model_version_trigger_by_model_iu ON models;
CREATE TRIGGER stream_model_version_trigger_by_model_iu
BEFORE UPDATE OF workspace_id ON models FOR EACH ROW EXECUTE PROCEDURE stream_model_version_change_by_model();

DROP TRIGGER IF EXISTS stream_model_version_trigger_d ON model_versions;
CREATE TRIGGER stream_model_version_trigger_d
    BEFORE DELETE ON model_versions
    FOR EACH ROW EXECUTE PROCEDURE stream_model_version_change();
