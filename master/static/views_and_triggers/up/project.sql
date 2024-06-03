CREATE FUNCTION stream_project_change() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        PERFORM stream_project_notify(NULL, to_jsonb(NEW));
    ELSEIF (TG_OP = 'UPDATE') THEN
        PERFORM stream_project_notify(to_jsonb(OLD), to_jsonb(NEW));
    ELSEIF (TG_OP = 'DELETE') THEN
        PERFORM stream_project_notify(to_jsonb(OLD), NULL);
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$;
CREATE TRIGGER stream_project_trigger_d BEFORE DELETE ON projects FOR EACH ROW EXECUTE PROCEDURE stream_project_change();
CREATE TRIGGER stream_project_trigger_iu AFTER INSERT OR UPDATE OF name, description, archived, created_at, notes, workspace_id, user_id, immutable, state ON projects FOR EACH ROW EXECUTE PROCEDURE stream_project_change();

CREATE FUNCTION stream_project_notify(before jsonb, after jsonb) RETURNS integer
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
    PERFORM pg_notify('stream_project_chan', output::text);
return 0;
END;
$$;

CREATE FUNCTION stream_project_seq_modify() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.seq = nextval('stream_project_seq');
RETURN NEW;
END;
$$;
CREATE TRIGGER stream_project_trigger_seq BEFORE INSERT OR UPDATE OF name, description, archived, created_at, notes, workspace_id, user_id, immutable, state ON projects FOR EACH ROW EXECUTE PROCEDURE stream_project_seq_modify();
