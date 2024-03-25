/*
 Rollback changing `partition_type` on `metrics` from ENUM type to TEXT.

 Partition keys cannot be modified without dropping and recreating the table.
 */

-- Detach partitions and drop parent `metrics` table.
ALTER TABLE metrics DETACH PARTITION raw_steps;
ALTER TABLE metrics DETACH PARTITION raw_validations;
ALTER TABLE metrics DETACH PARTITION generic_metrics;

DROP TABLE metrics;

-- Create `metric_partition_type` enum type.
CREATE TYPE metric_partition_type AS ENUM (
    'VALIDATION',
    'TRAINING',
    'GENERIC'
);

CREATE SEQUENCE metrics_id_seq;
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

-- Re-create table with `partition_type` ENUM.
CREATE TABLE metrics (
    trial_id integer NOT NULL,
    end_time timestamp with time zone,
    metrics jsonb,
    total_batches integer,
    trial_run_id integer NOT NULL DEFAULT 0,
    archived boolean NOT NULL DEFAULT false,
    id integer NOT NULL DEFAULT nextval('metrics_id_seq'),
    metric_group text,
    partition_type metric_partition_type NOT NULL DEFAULT 'GENERIC'
) PARTITION BY LIST (partition_type);

-- Modify child partitions to have `partition_type` ENUM and set defaults.
ALTER TABLE raw_steps ALTER COLUMN partition_type DROP DEFAULT;
ALTER TABLE raw_steps DROP CONSTRAINT partition_type_check;
ALTER TABLE raw_steps ALTER COLUMN partition_type TYPE metric_partition_type
    USING partition_type::metric_partition_type;
ALTER TABLE raw_steps ALTER COLUMN partition_type SET DEFAULT 'TRAINING';

ALTER TABLE raw_validations ALTER COLUMN partition_type DROP DEFAULT;
ALTER TABLE raw_validations DROP CONSTRAINT partition_type_check;
ALTER TABLE raw_validations ALTER COLUMN partition_type TYPE metric_partition_type
    USING partition_type::metric_partition_type;
ALTER TABLE raw_validations ALTER COLUMN partition_type SET DEFAULT 'VALIDATION';

ALTER TABLE generic_metrics ALTER COLUMN partition_type DROP DEFAULT;
ALTER TABLE generic_metrics DROP CONSTRAINT partition_type_check;
ALTER TABLE generic_metrics ALTER COLUMN partition_type TYPE metric_partition_type
    USING partition_type::metric_partition_type;
ALTER TABLE generic_metrics ALTER COLUMN partition_type SET DEFAULT 'GENERIC';


-- Reattach child partitions.
ALTER TABLE metrics ATTACH PARTITION generic_metrics FOR
    VALUES IN ('GENERIC');
ALTER TABLE metrics ATTACH PARTITION raw_validations FOR
    VALUES IN ('VALIDATION');
ALTER TABLE metrics ATTACH PARTITION raw_steps FOR
    VALUES IN ('TRAINING');
