ALTER TABLE public.users DROP COLUMN last_login;
ALTER TABLE public.users ALTER COLUMN modified_at TYPE timestamp;
