DROP TRIGGER IF EXISTS autoupdate_users_modified_at ON users;

CREATE TRIGGER autoupdate_users_modified_at
  BEFORE INSERT OR UPDATE OF password_hash, admin, active, display_name, remote ON public.users
  FOR EACH ROW
  EXECUTE PROCEDURE public.set_modified_time();
