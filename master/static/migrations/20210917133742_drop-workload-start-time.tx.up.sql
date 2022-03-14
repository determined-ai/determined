DROP VIEW steps;

ALTER TABLE public.raw_steps
    DROP COLUMN start_time;

ALTER TABLE public.raw_steps
    DROP COLUMN id;

ALTER TABLE public.raw_steps
    ADD COLUMN id SERIAL;

CREATE VIEW steps AS
    SELECT * FROM raw_steps WHERE NOT archived;

DROP VIEW validations;

ALTER TABLE public.raw_validations
    DROP COLUMN start_time CASCADE;

CREATE VIEW validations AS
    SELECT * FROM raw_validations WHERE NOT archived;

DROP VIEW checkpoints;

ALTER TABLE public.raw_checkpoints
    DROP COLUMN start_time;

CREATE VIEW checkpoints AS
    SELECT * FROM raw_checkpoints WHERE NOT archived;

