ALTER TABLE public.allocations
    ADD COLUMN state INT,
    ADD COLUMN is_ready BOOLEAN;
