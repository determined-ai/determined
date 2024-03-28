ALTER TABLE tasks
    DROP COLUMN log_retention_days;

DROP FUNCTION retention_timestamp;
