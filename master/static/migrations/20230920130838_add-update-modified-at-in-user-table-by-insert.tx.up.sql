CREATE TRIGGER autoupdate_users_modified_at_by_insert
  BEFORE INSERT ON public.users
  FOR EACH ROW
  EXECUTE PROCEDURE public.set_modified_time();
