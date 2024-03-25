/*
  Rollback adding a new partition for profiling metrics to `metrics`.
 */

-- Detach partition and drop table.
ALTER TABLE metrics DETACH PARTITION system_metrics;

DROP TABLE system_metrics CASCADE;
