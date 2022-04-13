WITH const AS (
    SELECT 
            $1 :: timestamptz
         AS target_date
),
oneday_agg AS (SELECT
    'imagepull' AS aggregation_type,
    row_to_json(row(resource_pool, 1)) AS aggregation_key,
    avg(
        extract(
            epoch
            FROM end_time - start_time
            ) 
    ) AS seconds
FROM task_stats, const
WHERE end_time <= const.target_date AND end_time > (const.target_date - interval '1 day') AND event_type = 'IMAGEPULL'
GROUP BY resource_pool),

sevenday_agg AS (SELECT
    'imagepull' AS aggregation_type,
    row_to_json(row(resource_pool, 7)) AS aggregation_key,
    avg(
        extract(
            epoch
            FROM end_time - start_time
            ) 
    ) AS seconds
FROM task_stats, const
WHERE end_time <= const.target_date AND end_time > (const.target_date - interval '7 days') AND event_type = 'IMAGEPULL'
GROUP BY resource_pool),

thirtyday_agg AS (SELECT
    'imagepull' AS aggregation_type,
    row_to_json(row(resource_pool, 30)) AS aggregation_key,
    avg(
        extract(
            epoch
            FROM end_time - start_time
            ) 
    ) AS seconds
FROM task_stats, const
WHERE end_time <= const.target_date AND end_time > (const.target_date - interval '30 days') AND event_type = 'IMAGEPULL'
GROUP BY resource_pool),

all_aggs AS (
    SELECT
        *
    FROM
        oneday_agg
    UNION ALL
    SELECT
        *
    FROM
        sevenday_agg
    UNION ALL
    SELECT
        *
    FROM
        thirtyday_agg
)

INSERT INTO
    resource_aggregates (
        SELECT
            const.target_date AS date,
            all_aggs.*
        FROM
            all_aggs, const
    )
    ON CONFLICT ON CONSTRAINT resource_aggregates_keys_unique
    DO UPDATE SET seconds = EXCLUDED.seconds
