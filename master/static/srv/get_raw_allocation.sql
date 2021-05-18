WITH const AS (
    SELECT
        tstzrange($1 :: timestamptz, $2 :: timestamptz) AS period
),
-- Workloads that had any overlap with the target interval, along with the length of the overlap of
-- their time with the requested period.
workloads AS (
    SELECT
        all_workloads.trial_id,
        all_workloads.kind,
        lower(all_workloads.range) AS start_time,
        upper(all_workloads.range) AS end_time,
        extract(
            epoch
            FROM
                -- `*` computes the intersection of the two ranges.
                upper(const.period * range) - lower(const.period * range)
        ) AS seconds
    FROM
        (
            -- Summarize the common relevant fields from all workload types. We might want this to
            -- be a CTE, but I think that would cause PostgreSQL <12 to insert an optimization fence
            -- and have to fully scan all three tables, which could be bad.
            SELECT
                'training' AS kind,
                trial_id,
                tstzrange(start_time, end_time) AS range
            FROM
                raw_steps
            UNION ALL
            SELECT
                'validation' AS kind,
                trial_id,
                tstzrange(start_time, end_time) AS range
            FROM
                raw_validations
            UNION ALL
            SELECT
                'checkpoint' AS kind,
                trial_id,
                tstzrange(start_time, end_time) AS range
            FROM
                raw_checkpoints
        ) AS all_workloads,
        const
    WHERE
        -- `&&` determines whether the ranges overlap.
        const.period && all_workloads.range
)
SELECT
    trials.experiment_id,
    workloads.kind,
    users.username,
    experiments.config -> 'resources' ->> 'slots_per_trial' AS slots,
    experiments.config -> 'labels' AS labels,
    workloads.start_time,
    workloads.end_time,
    workloads.seconds
FROM
    workloads,
    trials,
    experiments,
    users
WHERE
    workloads.trial_id = trials.id
    AND trials.experiment_id = experiments.id
    AND experiments.owner_id = users.id
ORDER BY
    start_time
