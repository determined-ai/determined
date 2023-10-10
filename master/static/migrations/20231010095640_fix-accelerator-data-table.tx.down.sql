ALTER TABLE public.allocation_accelerators
    DROP COLUMN id,
    ADD CONSTRAINT allocation_accelerators_pkey PRIMARY KEY(container_id); 
