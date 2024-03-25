/*
 Change `partition_type` on `metrics` from ENUM type to TEXT.

 Partition keys cannot be modified without dropping and recreating the table.
 */

-- Detach partitions and drop parent `metrics` table.
ALTER TABLE metrics DETACH PARTITION raw_steps;
ALTER TABLE metrics DETACH PARTITION raw_validations;
ALTER TABLE metrics DETACH PARTITION generic_metrics;

DROP TABLE metrics;

CREATE SEQUENCE IF NOT EXISTS metrics_id_seq;
SELECT setval(
    'metrics_id_seq',
    greatest(
        coalesce(
            (
                SELECT max(id)
                FROM raw_steps
            ),
            0
        ),
        coalesce(
            (
                SELECT max(id)
                FROM generic_metrics
            ),
            0
        ),
        coalesce(
            (
                SELECT max(id)
                FROM raw_validations
            ),
            0
        )
    ) + 1,
    true
);

-- Re-create table with `partition_type` TEXT.
CREATE TABLE metrics (
    trial_id integer NOT NULL,
    end_time timestamp with time zone,
    metrics jsonb,
    total_batches integer,
    trial_run_id integer NOT NULL DEFAULT 0,
    archived boolean NOT NULL DEFAULT false,
    id integer NOT NULL DEFAULT nextval('metrics_id_seq'),
    metric_group text,
    partition_type text NOT NULL DEFAULT 'GENERIC'
) PARTITION BY LIST (partition_type);


-- Modify child partitions to have `partition_type` TEXT and set defaults.
ALTER TABLE raw_steps ALTER COLUMN partition_type TYPE text;
ALTER TABLE raw_steps ALTER COLUMN partition_type SET DEFAULT 'TRAINING';

ALTER TABLE raw_validations ALTER COLUMN partition_type TYPE text;
ALTER TABLE raw_validations ALTER COLUMN partition_type SET DEFAULT 'VALIDATION';

ALTER TABLE generic_metrics ALTER COLUMN partition_type TYPE text;
ALTER TABLE generic_metrics ALTER COLUMN partition_type SET DEFAULT 'GENERIC';


-- Drop `metric_partition_type` enum.
DROP TYPE IF EXISTS metric_partition_type;


-- Add CHECK constraint on `partition_type` to child partitions
-- (this will speed up attaching partitions).
ALTER TABLE raw_steps ADD CONSTRAINT partition_type_check CHECK (partition_type='TRAINING'::text);
ALTER TABLE raw_validations ADD CONSTRAINT partition_type_check CHECK (partition_type='VALIDATION'::text);
ALTER TABLE generic_metrics ADD CONSTRAINT partition_type_check CHECK (partition_type='GENERIC'::text);


-- Reattach partitions.
ALTER TABLE metrics ATTACH PARTITION generic_metrics FOR
    VALUES IN ('GENERIC');
ALTER TABLE metrics ATTACH PARTITION raw_validations FOR
    VALUES IN ('VALIDATION');
ALTER TABLE metrics ATTACH PARTITION raw_steps FOR
    VALUES IN ('TRAINING');
