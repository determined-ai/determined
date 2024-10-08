WITH const AS (
    SELECT
        $1::timestamptz
        AS target_date
),
relevant_tasks AS (
    SELECT 
        tasks.task_id, 
        allocations.allocation_id, 
        allocations.resource_pool, 
        task_stats.event_type,
        task_stats.start_time,
        task_stats.end_time,
        const.target_date
    FROM tasks
    CROSS JOIN const
    INNER JOIN 
        allocations ON tasks.task_id = allocations.task_id
    INNER JOIN 
        task_stats ON allocations.allocation_id = task_stats.allocation_id
    WHERE
        -- Exclude the rows with NULL start_time. When Bun sees StartTime is nil,
        -- it saves it as 0001-01-01 00:00:00+00.
        task_stats.start_time != '0001-01-01 00:00:00+00:00'::TIMESTAMPTZ
        AND task_stats.end_time >= const.target_date
        AND task_stats.end_time < (const.target_date + interval '1 day')
        AND task_type in (
            'TRIAL',
            'NOTEBOOK',
            'SHELL',
            'COMMAND',
            'TENSORBOARD',
            'GENERIC'
        )
        AND task_stats.event_type = 'QUEUED'
),
day_agg AS (
    SELECT
        'queued' AS aggregation_type,
        relevant_tasks.resource_pool AS aggregation_key,
        avg(
            extract(
                EPOCH
                FROM relevant_tasks.end_time - relevant_tasks.start_time
            )
        ) AS seconds
    FROM relevant_tasks
    GROUP BY relevant_tasks.resource_pool
),
total_agg AS (
    SELECT
        'queued' AS aggregation_type,
        'total' AS aggregation_key,
        coalesce(avg(
            extract(
                EPOCH
                FROM relevant_tasks.end_time - relevant_tasks.start_time
            )
        ), 0) AS seconds
    FROM relevant_tasks
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
