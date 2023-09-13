 CREATE TABLE public.allocation_accelerators (
    container_id text NOT NULL PRIMARY KEY,
    allocation_id text NOT NULL REFERENCES public.allocations(allocation_id) ON DELETE CASCADE,
    node_name text NOT NULL,
    accelerator_type text NOT NULL,
    accelerator_uuids text []
);
