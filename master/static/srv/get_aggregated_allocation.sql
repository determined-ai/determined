WITH const AS (
    SELECT
        daterange($1 :: date, $2 :: date, '[]') AS period
)
SELECT
    to_char(resource_aggregates.date :: date, 'YYYY-MM-DD') AS date,
    -- These values must be kept in sync with the protobuf type ResourceAllocationAggregationType.
    CASE
        WHEN resource_aggregates.aggregation_type = 'total' THEN 'RESOURCE_ALLOCATION_AGGREGATION_TYPE_TOTAL'
        WHEN resource_aggregates.aggregation_type = 'user' THEN 'RESOURCE_ALLOCATION_AGGREGATION_TYPE_USER'
        WHEN resource_aggregates.aggregation_type = 'label' THEN 'RESOURCE_ALLOCATION_AGGREGATION_TYPE_LABEL'
        ELSE 'RESOURCE_ALLOCATION_AGGREGATION_TYPE_UNSPECIFIED'
    END AS aggregation_type,
    resource_aggregates.aggregation_key,
    seconds
FROM
    resource_aggregates,
    const
WHERE
    -- `@>` determines whether the range contains the time.
    const.period @> resource_aggregates.date
ORDER BY
    date
