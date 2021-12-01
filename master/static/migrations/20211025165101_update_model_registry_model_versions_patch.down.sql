ALTER TABLE public.model_versions ADD COLUMN readme text;
ALTER TABLE public.model_versions DROP COLUMN labels;
