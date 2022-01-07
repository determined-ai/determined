WITH const AS (
    SELECT
        tstzrange($1 :: timestamptz, $2 :: timestamptz) AS period
),
-- Workloads that had any overlap with the target interval, along with the length of the overlap of
-- their time with the requested period.
overlapping_allocs AS (
    SELECT
        *,
        tstzrange(start_time, end_time) AS range
    FROM allocations a, const
    WHERE
        -- `&&` determines whether the ranges overlap.
        const.period && tstzrange(a.start_time, a.end_time)
)
SELECT
    coalesce(u.username, 'unattributed'),
    a.slots,
    -- j.labels, -- Labels to be added back soon.
    a.start_time,
    a.end_time,
    extract(
        epoch
        FROM
            -- `*` computes the intersection of the two ranges.
            upper((SELECT period FROM const) * a.range)
            - lower((SELECT period FROM const) * a.range)
    ) AS seconds
FROM overlapping_allocs a
LEFT JOIN tasks t ON a.task_id = t.task_id
LEFT JOIN jobs j ON t.job_id = j.job_id
LEFT JOIN users u ON j.owner_id = u.id
ORDER BY start_time
