/*
 Allow NULL values on `metrics.total_batches`.

 Since `metrics` is a parent table that is partitioned into child tables, allow each child table
 to define whether or not it requires `total_batches` to be defined.
 */

-- On parent `metrics` table: drop NOT NULL and default on `total_batches`.
ALTER TABLE metrics ALTER COLUMN total_batches DROP NOT NULL;
ALTER TABLE metrics ALTER COLUMN total_batches DROP DEFAULT;


-- Add NOT NULL and default back to child partitions.
ALTER TABLE generic_metrics ALTER COLUMN total_batches SET NOT NULL;
ALTER TABLE generic_metrics ALTER COLUMN total_batches SET DEFAULT 0;

ALTER TABLE raw_validations ALTER COLUMN total_batches SET NOT NULL;
ALTER TABLE raw_validations ALTER COLUMN total_batches SET DEFAULT 0;

ALTER TABLE raw_steps ALTER COLUMN total_batches SET NOT NULL;
ALTER TABLE raw_steps ALTER COLUMN total_batches SET DEFAULT 0;

