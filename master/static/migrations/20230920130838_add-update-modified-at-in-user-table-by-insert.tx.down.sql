DROP TRIGGER IF EXISTS autoupdate_users_modified_at on users;

CREATE TRIGGER autoupdate_users_modified_at
  BEFORE INSERT ON public.users
  FOR EACH ROW
  EXECUTE PROCEDURE public.set_modified_time();
