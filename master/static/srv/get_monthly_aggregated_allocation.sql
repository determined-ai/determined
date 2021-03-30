WITH const AS (
    SELECT
        daterange($1 :: date, $2 :: date, '[]') AS period
),
months AS (
    SELECT
        date_trunc('month', resource_aggregates.date :: date) AT time zone 'UTC' AS period_start,
        aggregation_type,
        resource_aggregates.aggregation_key,
        sum(seconds) AS seconds
    FROM
        resource_aggregates,
        const
    WHERE
        -- `@>` determines whether the range contains the time.
        const.period @> resource_aggregates.date
    GROUP BY
        date_trunc('month', resource_aggregates.date :: date) AT time zone 'UTC',
        resource_aggregates.aggregation_type,
        resource_aggregates.aggregation_key
),
starts AS (
    SELECT
        DISTINCT(period_start) AS period_start
    FROM
        months
)
SELECT
    to_char(period_start, 'YYYY-MM') AS period_start,
    'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY' AS period,
    (
        SELECT
            seconds
        FROM
            months
        WHERE
            aggregation_type = 'total'
            AND months.period_start = starts.period_start
        LIMIT
            1
    ) AS seconds,
    (
        SELECT
            jsonb_object_agg(aggregation_key, seconds)
        FROM
            months
        WHERE
            aggregation_type = 'username'
            AND months.period_start = starts.period_start
    ) AS by_username,
    (
        SELECT
            jsonb_object_agg(aggregation_key, seconds)
        FROM
            months
        WHERE
            aggregation_type = 'experiment_label'
            AND months.period_start = starts.period_start
    ) AS by_experiment_label,
    (
        SELECT
            jsonb_object_agg(aggregation_key, seconds)
        FROM
            months
        WHERE
            aggregation_type = 'resource_pool'
            AND months.period_start = starts.period_start
    ) AS by_resource_pool,
    (
        SELECT
            jsonb_object_agg(aggregation_key, seconds)
        FROM
            months
        WHERE
            aggregation_type = 'agent_label'
            AND months.period_start = starts.period_start
    ) AS by_agent_label
FROM
    starts
ORDER BY
    period_start
