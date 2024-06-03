CREATE OR REPLACE FUNCTION stream_project_change() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        PERFORM stream_project_notify(
            NULL, jsonb_build_object('id', NEW.id, 'workspace_id', NEW.workspace_id, 'seq', NEW.seq)
        );
    ELSEIF (TG_OP = 'UPDATE') THEN
        PERFORM stream_project_notify(
            jsonb_build_object('id', OLD.id, 'workspace_id', OLD.workspace_id, 'seq', OLD.seq), 
            jsonb_build_object('id', NEW.id, 'workspace_id', NEW.workspace_id, 'seq', NEW.seq)
        );
    ELSEIF (TG_OP = 'DELETE') THEN
        PERFORM stream_project_notify(
            jsonb_build_object('id', OLD.id, 'workspace_id', OLD.workspace_id, 'seq', OLD.seq), NULL
        );
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION stream_model_change() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        PERFORM stream_model_notify(
            NULL, jsonb_build_object('id', NEW.id, 'workspace_id', NEW.workspace_id, 'seq', NEW.seq)
        );
    ELSEIF (TG_OP = 'UPDATE') THEN
        PERFORM stream_model_notify(
            jsonb_build_object('id', OLD.id, 'workspace_id', OLD.workspace_id, 'seq', OLD.seq), 
            jsonb_build_object('id', NEW.id, 'workspace_id', NEW.workspace_id, 'seq', NEW.seq)
        );
    ELSEIF (TG_OP = 'DELETE') THEN
        PERFORM stream_model_notify(
            jsonb_build_object('id', OLD.id, 'workspace_id', OLD.workspace_id, 'seq', OLD.seq), NULL
        );
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION stream_model_version_change() RETURNS TRIGGER AS $$
DECLARE
    n jsonb = NULL;
    o jsonb = NULL;
BEGIN 
    IF (TG_OP = 'INSERT') THEN
        n = jsonb_set('id', NEW.id, '{workspace_id}', to_jsonb((SELECT workspace_id FROM models WHERE id = NEW.model_id)::text));
        PERFORM stream_model_version_notify(NULL, n);
    ELSEIF (TG_OP = 'UPDATE') THEN
        o = jsonb_set('id', OLD.id,, '{workspace_id}', to_jsonb((SELECT workspace_id FROM models WHERE id = OLD.model_id)::text));
        n = jsonb_set('id', NEW.id, '{workspace_id}', to_jsonb((SELECT workspace_id FROM models WHERE id = NEW.model_id)::text));
        PERFORM stream_model_version_notify(o, n);
    ELSEIF (TG_OP = 'DELETE') THEN
        PERFORM stream_model_version_notify(jsonb_build_object('id', OLD.id), NULL);
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
        PERFORM stream_model_version_notify(
            jsonb_set('id', f.id, '{workspace_id}', to_jsonb(OLD.workspace_id::text)), 
            jsonb_set('id', f.id, '{workspace_id}', to_jsonb(NEW.workspace_id::text)));
    END LOOP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;