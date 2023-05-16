UPDATE public.experiments SET notes = '' WHERE notes IS NULL;

ALTER TABLE public.experiments ALTER COLUMN notes SET NOT NULL;
