CREATE TABLE public.allocation_accelerators (
    container_id text NOT NULL PRIMARY KEY,
    allocation_id text NOT NULL REFERENCES public.allocations(allocation_id),
    node_name text NOT NULL,
    accelerator_type text NOT NULL,
    accelerators text []
);
