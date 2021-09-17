ALTER TABLE public.raw_steps
    DROP COLUMN start_time CASCADE; -- Cascade to the view destroys it.

ALTER TABLE public.raw_steps
    DROP COLUMN id CASCADE; -- Should be nothing to cascade to at this point but :shrug:

ALTER TABLE public.raw_steps
    ADD COLUMN id SERIAL;

CREATE OR REPLACE VIEW steps AS -- But 'OR REPLACE' anyway, just in case :shrug:
    SELECT * FROM raw_steps WHERE NOT archived;

ALTER TABLE public.raw_validations
    DROP COLUMN start_time CASCADE;

CREATE OR REPLACE VIEW validations AS
    SELECT * FROM raw_validations WHERE NOT archived;

ALTER TABLE public.raw_checkpoints
    DROP COLUMN start_time CASCADE;

CREATE OR REPLACE VIEW checkpoints AS
    SELECT * FROM raw_checkpoints WHERE NOT archived;

