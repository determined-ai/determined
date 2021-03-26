WITH const AS (
    SELECT
        daterange($1 :: date, $2 :: date, '[]') AS period
),
days AS (
    SELECT
        resource_aggregates.date :: date AS period_start,
        aggregation_type,
        resource_aggregates.aggregation_key,
        seconds
    FROM
        resource_aggregates,
        const
    WHERE
        -- `@>` determines whether the range contains the time.
        const.period @> resource_aggregates.date
),
starts AS (
    SELECT
        DISTINCT(period_start) AS period_start
    FROM
        days
)
SELECT
    to_char(period_start, 'YYYY-MM-DD') AS period_start,
    'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY' AS period,
    (
        SELECT
            seconds
        FROM
            days
        WHERE
            aggregation_type = 'total'
            AND days.period_start = starts.period_start
        LIMIT
            1
    ) AS seconds,
    (
        SELECT
            jsonb_object_agg(aggregation_key, seconds)
        FROM
            days
        WHERE
            aggregation_type = 'user'
            AND days.period_start = starts.period_start
    ) AS by_user,
    (
        SELECT
            jsonb_object_agg(aggregation_key, seconds)
        FROM
            days
        WHERE
            aggregation_type = 'label'
            AND days.period_start = starts.period_start
    ) AS by_label
FROM
    starts
ORDER BY
    period_start
