WITH const AS (
    SELECT
        $1 :: timestamp AS day_start,
        $2 :: timestamp AS day_end
),
-- Workloads that had any overlap with the target interval, with their start and end times bounded
-- to the interval.
workloads AS (
    SELECT
        all_workloads.trial_id,
        all_workloads.kind,
        all_workloads.start_time,
        all_workloads.end_time,
        extract(
            epoch
            FROM
                CASE
                    WHEN all_workloads.end_time IS NULL THEN const.day_end
                    WHEN all_workloads.end_time > const.day_end THEN const.day_end
                    ELSE all_workloads.end_time
                END - CASE
                    WHEN all_workloads.start_time < const.day_start THEN const.day_start
                    ELSE all_workloads.start_time
                END
        ) AS seconds
    FROM
        (
            -- Summarize the common relevant fields from all workload types. We might want this to
            -- be a CTE, but I think that would cause PostgreSQL <12 to insert an optimization fence
            -- and have to fully scan all three tables, which could be bad.
            SELECT
                'step' AS kind,
                trial_id,
                start_time,
                end_time
            FROM
                steps
            UNION ALL
            SELECT
                'validation' AS kind,
                trial_id,
                start_time,
                end_time
            FROM
                validations
            UNION ALL
            SELECT
                'checkpoint' AS kind,
                trial_id,
                start_time,
                end_time
            FROM
                checkpoints
        ) AS all_workloads,
        const
    WHERE
        all_workloads.start_time <= const.day_end
        AND coalesce(all_workloads.end_time, const.day_end) >= const.day_start
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
