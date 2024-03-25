/*
 Rollback allowing NULL values on `metrics.total_batches`.

 Drops NOT NULL and default values on `total_batches` for child partitions of `metrics`.
 Adds a global NOT NULL and default value back to `metrics` table to be inherited by all children.
 */


-- Drop NOT NULL and default for `total_batches` on child partitions.
ALTER TABLE generic_metrics ALTER COLUMN total_batches DROP NOT NULL;
ALTER TABLE generic_metrics ALTER COLUMN total_batches DROP DEFAULT;

ALTER TABLE raw_validations ALTER COLUMN total_batches DROP NOT NULL;
ALTER TABLE raw_validations ALTER COLUMN total_batches DROP DEFAULT;

ALTER TABLE raw_steps ALTER COLUMN total_batches DROP NOT NULL;
ALTER TABLE raw_steps ALTER COLUMN total_batches DROP DEFAULT;

-- Add NOT NULL and default on `total_batches` to parent `metrics` table.
ALTER TABLE metrics ALTER COLUMN total_batches SET NOT NULL;
ALTER TABLE metrics ALTER COLUMN total_batches SET DEFAULT 0;


