ALTER TABLE public.model_versions DROP COLUMN readme;
ALTER TABLE public.model_versions ADD COLUMN labels text[];
