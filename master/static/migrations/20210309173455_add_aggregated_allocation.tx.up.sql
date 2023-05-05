CREATE TABLE public.resource_aggregates (
    date date NOT NULL,
    aggregation_type text NOT NULL,
    aggregation_key text NOT NULL,
    seconds float NOT NULL
);
ALTER TABLE
    public.resource_aggregates
ADD
    CONSTRAINT resource_aggregates_keys_unique UNIQUE (date, aggregation_type, aggregation_key);
