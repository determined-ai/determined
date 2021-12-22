ALTER TABLE public.allocations
    ADD COLUMN state INT;

ALTER TABLE public.allocations
    ADD COLUMN isReady BOOLEAN;
