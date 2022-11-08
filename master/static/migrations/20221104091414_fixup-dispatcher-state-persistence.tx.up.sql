ALTER TABLE public.resourcemanagers_dispatcher_dispatches ALTER COLUMN resource_id DROP NOT NULL;
ALTER TABLE public.resourcemanagers_dispatcher_dispatches
  DROP CONSTRAINT resourcemanagers_dispatcher_dispatches_resource_id_fkey,
  ADD CONSTRAINT resourcemanagers_dispatcher_dispatches_resource_id_fkey
    FOREIGN KEY (resource_id) REFERENCES public.allocation_resources(resource_id) ON DELETE SET NULL;
