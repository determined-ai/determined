ALTER TABLE public.checkpoints ADD COLUMN format character varying;
ALTER TABLE public.checkpoints ADD COLUMN framework character varying;
ALTER TABLE public.checkpoints ADD COLUMN determined_version character varying;
