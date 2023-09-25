CREATE OR REPLACE FUNCTION public.set_modified_time ()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF (TG_OP = 'INSERT' OR TG_OP = 'UPDATE') THEN
        NEW.modified_at := now();
        RETURN NEW;
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS autoupdate_users_modified_at on users;

CREATE TRIGGER autoupdate_users_modified_at
  BEFORE INSERT OR UPDATE ON public.users
  FOR EACH ROW
  EXECUTE PROCEDURE public.set_modified_time();
