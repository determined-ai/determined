ALTER TABLE public.allocations
    ADD COLUMN state INT,
    ADD COLUMN isReady BOOLEAN;
