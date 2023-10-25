DROP TRIGGER IF EXISTS autoupdate_users_modified_at ON users;

CREATE OR REPLACE FUNCTION public.set_modified_time ()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.modified_at := now();
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_users_modified_at
  BEFORE INSERT OR UPDATE OF username, password_hash, admin, active, display_name, remote ON public.users
  FOR EACH ROW
  EXECUTE PROCEDURE public.set_modified_time();
