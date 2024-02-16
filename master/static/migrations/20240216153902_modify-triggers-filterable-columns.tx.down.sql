-- return original definition of stream_project_change
CREATE OR REPLACE FUNCTION stream_project_change() RETURNS TRIGGER AS $$
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
$$ LANGUAGE plpgsql;
