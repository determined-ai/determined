/*
  Add a new partition for profiling metrics to `metrics`.
 */

-- Create new `system_metrics` table.
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

CREATE TABLE system_metrics (
    trial_id integer NOT NULL,
    end_time timestamp with time zone,
    metrics jsonb,
    total_batches integer,
    trial_run_id integer NOT NULL DEFAULT 0,
    archived boolean NOT NULL DEFAULT false,
    id integer NOT NULL DEFAULT nextval('metrics_id_seq'),
    partition_type text NOT NULL DEFAULT 'PROFILING',
    metric_group text,
    FOREIGN KEY (trial_id) REFERENCES runs (id)
);
-- Add CHECK constraint on `partition_type` (this will speed up attaching partitions).
ALTER TABLE system_metrics ADD CONSTRAINT partition_type_check CHECK (partition_type='PROFILING'::text);

-- Attach table as partition to `metrics`.
ALTER TABLE metrics ATTACH PARTITION system_metrics FOR
    VALUES IN ('PROFILING');
