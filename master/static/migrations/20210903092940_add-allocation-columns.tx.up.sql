ALTER TABLE public.allocations
    ADD COLUMN slots smallint NOT NULL DEFAULT 1,
    ADD COLUMN agent_label text NOT NULL DEFAULT '';
