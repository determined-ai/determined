ALTER TABLE tasks
    ADD COLUMN log_retention_days SMALLINT;

-- This function is used for testing purposes, it should return the current timestamp
-- under normal circumstances. However, it can be overridden in tests to simulate
-- different timestamps.
CREATE FUNCTION retention_timestamp() RETURNS TIMESTAMPTZ AS $$
    BEGIN
        RETURN transaction_timestamp();
    END
    $$ LANGUAGE PLPGSQL;
