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
USING (CASE state
    WHEN 0 THEN 'PENDING'
    WHEN 1 THEN 'ASSIGNED'
    WHEN 2 THEN 'PULLING'
    WHEN 3 THEN 'STARTING'
    WHEN 4 THEN 'RUNNING'
    WHEN 5 THEN 'TERMINATING'
    WHEN 6 THEN 'TERMINATED'
END)::public.allocation_state;    
