WITH const AS (
    SELECT
        tstzrange(
            $1 :: timestamptz,
            ($1 :: timestamptz + interval '1 day')
        ) AS period
),
-- Allocations that had any overlap with the target interval, along with the length of the overlap of
-- their time with the requested period.
allocs_in_range AS (
    SELECT
        a.*,
        extract(
            epoch
            FROM
                -- `*` computes the intersection of the two ranges.
                upper(const.period * a.range) - lower(const.period * a.range)
        ) * a.slots :: float AS seconds
    FROM
        (
            SELECT
                *,
                tstzrange(start_time, greatest(start_time, end_time)) AS range
            FROM
                allocations
            WHERE
                start_time IS NOT NULL
        ) AS a,
        const
    WHERE
        -- `&&` determines whether the ranges overlap.
        const.period && a.range
),
user_agg AS (
    SELECT
        'username' AS aggregation_type,
        users.username AS aggregation_key,
        sum(allocs_in_range.seconds) AS seconds
    FROM
        allocs_in_range,
        tasks,
        jobs,
        -- Since a job is a user submission by definition, eventually user information
        -- should live in the generic job table and this can use that.
        users
    WHERE
        allocs_in_range.task_id = tasks.task_id
        AND tasks.job_id = jobs.job_id
        AND jobs.owner_id = users.id
    GROUP BY
        users.username
),
label_agg AS (
    SELECT
        'experiment_label' AS aggregation_type,
        -- This seems to be the most convenient way to convert from a JSONB string value to a normal
        -- string value.
        labels.label #>> '{}' AS aggregation_key,
        sum(allocs_in_range.seconds) AS seconds
    FROM
        allocs_in_range,
        trials,
        -- An exploded view of experiment labels (one row for each label for each experiment).
        -- If we want this to work for generic jobs, we will likely need to rethink labels.
        (
            SELECT
                id,
                jsonb_array_elements(
                    CASE
                        WHEN config ->> 'labels' IS NULL THEN '[]' :: jsonb
                        ELSE config -> 'labels'
                    END
                ) AS label
            FROM
                experiments
        ) AS labels
    WHERE
        allocs_in_range.task_id = trials.task_id
        AND trials.experiment_id = labels.id
    GROUP BY
        labels.label
),
pool_agg AS (
    SELECT
        'resource_pool' AS aggregation_type,
        allocs_in_range.resource_pool,
        sum(allocs_in_range.seconds) AS seconds
    FROM
        allocs_in_range
    GROUP BY
        allocs_in_range.resource_pool
),
all_aggs AS (
    SELECT
        *
    FROM
        user_agg
    UNION ALL
    SELECT
        *
    FROM
        label_agg
    UNION ALL
    SELECT
        *
    FROM
        pool_agg
    UNION ALL
    SELECT
        'total' AS aggregation_type,
        'total' AS aggregation_key,
        coalesce(sum(allocs_in_range.seconds), 0) AS seconds
    FROM
        allocs_in_range
)
INSERT INTO
    resource_aggregates (
        SELECT
            lower(const.period) AS date,
            all_aggs.*
        FROM
            all_aggs,
            const
    )
