WITH const AS (
    SELECT tstzrange($1::timestamptz, $2::timestamptz) AS period
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
            EPOCH
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
                kind,
                trial_id,
                tstzrange(start_time, end_time) AS range
            FROM
                (
                    SELECT
                        kind,
                        trial_id,
                        -- Here lies an implicit assumption that one workload started when the previous ended.
                        lag(end_time, 1) OVER (
                            PARTITION BY trial_id
                            ORDER BY
                                end_time
                        ) AS start_time,
                        end_time
                    FROM
                        (
                            -- Start of first is the start of the trial
                            SELECT
                                NULL AS kind,
                                id AS trial_id,
                                start_time AS end_time
                            FROM
                                trials
                            UNION ALL
                            -- Or more accurately of late, start of the allocation.
                            SELECT
                                NULL AS kind,
                                tr.id AS trial_id,
                                a.start_time AS end_time
                            FROM
                                allocations a,
                                tasks t,
                                trials tr
                            WHERE
                                a.task_id = t.task_id
                                AND t.task_id = tr.task_id
                            UNION ALL
                            SELECT
                                'training' AS kind,
                                trial_id,
                                end_time
                            FROM
                                raw_steps
                            UNION ALL
                            SELECT
                                'validation' AS kind,
                                trial_id,
                                end_time
                            FROM
                                raw_validations
                            UNION ALL
                            SELECT
                                'checkpointing' AS kind,
                                t.id AS trial_id,
                                checkpoints_v2.report_time AS end_time
                            FROM
                                checkpoints_v2
                            JOIN
                                trials AS t ON checkpoints_v2.task_id = t.task_id
                            UNION ALL
                            SELECT
                                'imagepulling' AS kind,
                                trials.id,
                                task_stats.end_time
                            FROM
                                task_stats, trials, allocations
                            WHERE
                                task_stats.event_type = 'IMAGEPULL'
                                AND allocations.allocation_id = task_stats.allocation_id
                                AND allocations.task_id = trials.task_id
                        ) metric_reports
                ) derived_workload_spans
            WHERE
                start_time IS NOT NULL
                AND end_time IS NOT NULL
                AND kind IS NOT NULL
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
    experiments.owner_id AS user_id,
    (experiments.config -> 'resources' ->> 'slots_per_trial')::smallint AS slots,
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
UNION
SELECT
    NULL AS experiment_id,
    'agent' AS kind,
    agent_id AS username,
    NULL AS user_id,
    slots,
    NULL AS labels,
    start_time,
    end_time,
    extract(
        EPOCH
        FROM
        -- `*` computes the intersection of the two ranges.
        upper(const.period * tstzrange(start_time, end_time))
        - lower(const.period * tstzrange(start_time, end_time))
    ) AS seconds
FROM
    agent_stats, const
WHERE const.period && tstzrange(start_time, end_time)
UNION
SELECT
    NULL AS experiment_id,
    'instance' AS kind,
    instance_id AS username,
    NULL AS user_id,
    slots,
    NULL AS labels,
    start_time,
    end_time,
    extract(
        EPOCH
        FROM
        -- `*` computes the intersection of the two ranges.
        upper(const.period * tstzrange(start_time, end_time))
        - lower(const.period * tstzrange(start_time, end_time))
    ) AS seconds
FROM
    provisioner_instance_stats, const
WHERE const.period && tstzrange(start_time, end_time)
ORDER BY
    start_time
