WITH const AS (
    SELECT
        $1::timestamptz
        AS target_date
),

day_agg AS (
    SELECT
        'queued' AS aggregation_type,
        allocations.resource_pool AS aggregation_key,
        avg(
            extract(
                EPOCH
                FROM task_stats.end_time - task_stats.start_time
            )
        ) AS seconds
    FROM task_stats, const, allocations
    WHERE
        allocations.allocation_id = task_stats.allocation_id
        -- Exclude the rows with NULL start_time. When Bun sees StartTime is nil,
        -- it saves it as 0001-01-01 00:00:00+00.
        AND task_stats.start_time != '0001-01-01 00:00:00+00:00'::TIMESTAMPTZ
        AND task_stats.end_time >= const.target_date
        AND task_stats.end_time < (const.target_date + interval '1 day')
        AND event_type = 'QUEUED'
    GROUP BY allocations.resource_pool
),

total_agg AS (
    SELECT
        'queued' AS aggregation_type,
        'total' AS aggregation_key,
        coalesce(avg(
            extract(
                EPOCH
                FROM end_time - start_time
            )
        ), 0) AS seconds
    FROM task_stats, const
    WHERE
        end_time >= const.target_date
        AND end_time < (const.target_date + interval '1 day')
        AND event_type = 'QUEUED'
        AND task_stats.start_time != '0001-01-01 00:00:00+00:00'::TIMESTAMPTZ
),

all_aggs AS (
    SELECT *
    FROM
        day_agg
    UNION ALL
    SELECT *
    FROM
        total_agg
)

INSERT INTO
resource_aggregates (
    SELECT
        timezone('Etc/UTC', const.target_date) AS date,
        all_aggs.*
    FROM
        all_aggs, const
)
ON CONFLICT ON CONSTRAINT resource_aggregates_keys_unique
DO UPDATE SET seconds = excluded.seconds
