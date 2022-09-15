ALTER TABLE public.allocations
ALTER COLUMN state
SET DATA TYPE VARCHAR(255);

DROP TYPE public.allocation_state;

CREATE TYPE public.allocation_state as ENUM (
    'PENDING',
    'ASSIGNED',
    'PULLING',
    'STARTING',
    'RUNNING',
    'TERMINATING',
    'TERMINATED'
);

ALTER TABLE public.allocations
ALTER COLUMN state
SET DATA TYPE public.allocation_state
USING state::public.allocation_state;