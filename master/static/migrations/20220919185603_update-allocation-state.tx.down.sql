ALTER TYPE public.allocation_state RENAME TO _allocation_state; 

CREATE TYPE public.allocation_state as ENUM (
    'PENDING',
    'ASSIGNED',
    'PULLING',
    'STARTING',
    'RUNNING',
    'TERMINATING',
    'TERMINATED'
);

DELETE FROM public.jobs WHERE job_type = 'WAITING'

ALTER TABLE public.allocations ALTER COLUMN state SET DATA TYPE public.allocation_state USING (state::text::allocation_state);

DROP TYPE public._allocation_state;