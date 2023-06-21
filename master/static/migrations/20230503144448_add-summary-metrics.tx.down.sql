ALTER TABLE trials
    DROP COLUMN summary_metrics,
    DROP COLUMN summary_metrics_timestamp;

DROP AGGREGATE IF EXISTS safe_sum(numeric);
