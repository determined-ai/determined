ALTER TABLE tasks
    ADD COLUMN log_retention_days SMALLINT;

CREATE FUNCTION retention_timestamp() RETURNS TIMESTAMPTZ AS $$
    BEGIN
        RETURN transaction_timestamp();
    END
    $$ LANGUAGE PLPGSQL;
