CREATE OR REPLACE FUNCTION public.set_modified_time ()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF (TG_OP = 'UPDATE') THEN
        NEW.modified_at := now();
        RETURN NEW;
    END IF;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_users_modified_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE PROCEDURE set_modified_time ();
