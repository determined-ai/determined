ALTER TABLE public.model_versions ADD COLUMN notes text;
ALTER TABLE public.models DROP COLUMN readme;
