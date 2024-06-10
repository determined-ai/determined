CREATE FUNCTION stream_model_change() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        PERFORM stream_model_notify(
            NULL, jsonb_build_object('id', NEW.id, 'workspace_id', NEW.workspace_id, 'user_id', NEW.user_id, 'seq', NEW.seq)
        );
    ELSEIF (TG_OP = 'UPDATE') THEN
        PERFORM stream_model_notify(
            jsonb_build_object('id', OLD.id, 'workspace_id', OLD.workspace_id, 'user_id', OLD.user_id, 'seq', OLD.seq),
            jsonb_build_object('id', NEW.id, 'workspace_id', NEW.workspace_id, 'user_id', NEW.user_id, 'seq', NEW.seq)
        );
    ELSEIF (TG_OP = 'DELETE') THEN
        PERFORM stream_model_notify(
            jsonb_build_object('id', OLD.id, 'workspace_id', OLD.workspace_id, 'user_id', OLD.user_id, 'seq', OLD.seq), NULL
        );
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$;
CREATE TRIGGER stream_model_trigger_d BEFORE DELETE ON models FOR EACH ROW EXECUTE PROCEDURE stream_model_change();
CREATE TRIGGER stream_model_trigger_iu AFTER INSERT OR UPDATE OF name, description, creation_time, last_updated_time, metadata, labels, user_id, archived, notes, workspace_id ON models FOR EACH ROW EXECUTE PROCEDURE stream_model_change();


CREATE FUNCTION stream_model_notify(before jsonb, after jsonb) RETURNS integer
    LANGUAGE plpgsql
    AS $$
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
$$;

CREATE FUNCTION stream_model_seq_modify() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.seq = nextval('stream_model_seq');
RETURN NEW;
END;
$$;
CREATE TRIGGER stream_model_trigger_seq BEFORE INSERT OR UPDATE OF name, description, creation_time, last_updated_time, metadata, labels, user_id, archived, notes, workspace_id ON models FOR EACH ROW EXECUTE PROCEDURE stream_model_seq_modify();


CREATE FUNCTION stream_model_version_change() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    n jsonb = NULL;
    o jsonb = NULL;
BEGIN
    IF (TG_OP = 'INSERT') THEN
        n = jsonb_build_object(
            'id', NEW.id,
            'workspace_id', to_jsonb((SELECT workspace_id FROM models WHERE id = NEW.model_id)::text),
            'model_id', NEW.model_id,
            'user_id', to_jsonb((SELECT user_id FROM models WHERE id = NEW.model_id)::integer),
            'seq', NEW.seq
        );
        PERFORM stream_model_version_notify(NULL, n);
    ELSEIF (TG_OP = 'UPDATE') THEN
        o = jsonb_build_object(
            'id', OLD.id,
            'workspace_id', to_jsonb((SELECT workspace_id FROM models WHERE id = OLD.model_id)::text),
            'model_id', OLD.model_id,
            'user_id', to_jsonb((SELECT user_id FROM models WHERE id = OLD.model_id)::integer),
            'seq', OLD.seq
        );
        n = jsonb_build_object(
            'id', NEW.id,
            'workspace_id', to_jsonb((SELECT workspace_id FROM models WHERE id = NEW.model_id)::text),
            'model_id', NEW.model_id,
            'user_id', to_jsonb((SELECT user_id FROM models WHERE id = NEW.model_id)::integer),
            'seq', NEW.seq
        );
        PERFORM stream_model_version_notify(o, n);
    ELSEIF (TG_OP = 'DELETE') THEN
        PERFORM stream_model_version_notify(jsonb_build_object('id', OLD.id), NULL);
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$;
CREATE TRIGGER stream_model_version_trigger_d BEFORE DELETE ON model_versions FOR EACH ROW EXECUTE PROCEDURE stream_model_version_change();
CREATE TRIGGER stream_model_version_trigger_iu AFTER INSERT OR UPDATE OF name, version, checkpoint_uuid, last_updated_time, metadata, labels, user_id, model_id, notes, comment ON model_versions FOR EACH ROW EXECUTE PROCEDURE stream_model_version_change();


CREATE FUNCTION stream_model_version_change_by_model() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    f record;
BEGIN
    FOR f in (SELECT * FROM model_versions WHERE model_id = NEW.id)
    LOOP
        PERFORM stream_model_version_notify(
            jsonb_build_object(
                'id', f.id,
                'workspace_id', to_jsonb(OLD.workspace_id::text),
                'model_id', f.model_id,
                'user_id', to_jsonb(OLD.user_id::integer),
                'seq', f.seq
            ),
            jsonb_build_object(
                'id', f.id,
                'workspace_id', to_jsonb(NEW.workspace_id::text),
                'model_id', f.model_id,
                'user_id', to_jsonb(NEW.user_id::integer),
                'seq', f.seq
            )
        );
    END LOOP;
    RETURN NEW;
END;
$$;
CREATE TRIGGER stream_model_version_trigger_by_model_iu BEFORE UPDATE OF workspace_id ON models FOR EACH ROW EXECUTE PROCEDURE stream_model_version_change_by_model();

CREATE FUNCTION stream_model_version_notify(before jsonb, after jsonb) RETURNS integer
    LANGUAGE plpgsql
    AS $$
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
$$;

CREATE FUNCTION stream_model_version_seq_modify() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.seq = nextval('stream_model_version_seq');
RETURN NEW;
END;
$$;
CREATE TRIGGER stream_model_version_trigger_seq BEFORE INSERT OR UPDATE OF name, version, checkpoint_uuid, last_updated_time, metadata, labels, user_id, model_id, notes, comment ON model_versions FOR EACH ROW EXECUTE PROCEDURE stream_model_version_seq_modify();

CREATE FUNCTION stream_model_version_seq_modify_by_model() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE model_versions SET seq = nextval('stream_model_version_seq') WHERE model_id = NEW.id;
RETURN NEW;
END;
$$;
CREATE TRIGGER stream_model_version_trigger_by_model BEFORE UPDATE OF workspace_id ON models FOR EACH ROW EXECUTE PROCEDURE stream_model_version_seq_modify_by_model();
