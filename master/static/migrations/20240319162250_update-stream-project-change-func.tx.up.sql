-- Trigger function to NOTIFY the master of changes, for INSERT/UPDATE/DELETE.
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