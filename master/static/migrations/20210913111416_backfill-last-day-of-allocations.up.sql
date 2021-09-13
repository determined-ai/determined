UPDATE
    trials
SET
    task_id = 'backported.' || id :: text;

INSERT INTO
    tasks (
        SELECT
            'backported.' || t.id :: text AS task_id,
            'TRIAL' AS task_type,
            -- Tasks are inserted when trial_id is not set, so we won't conflict trials with steps.
            t.start_time AS start_time
        FROM
            trials t
    );

WITH today AS (
    SELECT
        date_trunc('day', current_timestamp AT TIME ZONE 'UTC') AS ts
),
const AS (
    SELECT
        tstzrange(today.ts, (today.ts + interval '1 day')) AS period
    FROM
        today
)
INSERT INTO
    allocations (
        SELECT
            -- Make the trial ID _some_ predefined well know string so we can link public.trials easily.
            'backported.' || t.id :: text AS task_id,
            'backported.' || t.id :: text || '.' || all_workloads.kind || '.' || all_workloads.id :: text AS allocation_id,
            coalesce(
                e.config #>> '{resources, resource_pool}',
                'default'
            ) AS resource_pool,
            lower(const.period * range) AS start_time,
            upper(const.period * range) AS end_time,
            (e.config -> 'resources' ->> 'slots_per_trial') :: smallint AS slots,
            coalesce(e.config #>> '{resources, agent_label}', '') AS agent_label
        FROM
            (
                -- We could aggregate these to a single allocation per trial, but an 'allocation' per step
                -- works just fine (as far as the rollup knows, this could be true).
                SELECT
                    id,
                    't' AS kind,
                    trial_id,
                    tstzrange(start_time, end_time) AS range
                FROM
                    steps
                UNION ALL
                SELECT
                    id,
                    'v' AS kind,
                    trial_id,
                    tstzrange(start_time, end_time) AS range
                FROM
                    validations
                UNION ALL
                SELECT
                    id,
                    'c' AS kind,
                    trial_id,
                    tstzrange(start_time, end_time) AS range
                FROM
                    checkpoints
            ) AS all_workloads,
            trials t,
            experiments e,
            const
        WHERE
            const.period && all_workloads.range
            AND all_workloads.trial_id = t.id
            AND t.experiment_id = e.id
    );
