ALTER TABLE public.allocations
    ADD COLUMN exit_reason text,
    ADD COLUMN exit_error text,
    ADD COLUMN status_code int;
