WITH const AS (
    SELECT
        tstzrange(
            $1 :: timestamptz,
            ($1 :: timestamptz + interval '1 day')
        ) AS period
),
-- Workloads that had any overlap with the target interval, along with the length of the overlap of
-- their time with the requested period.
workloads AS (
    SELECT
        all_workloads.trial_id,
        extract(
            epoch
            FROM
                -- `*` computes the intersection of the two ranges.
                upper(const.period * range) - lower(const.period * range)
        ) AS seconds
    FROM
        (
            SELECT
                trial_id,
                tstzrange(start_time, end_time) AS range
            FROM
                steps
            UNION ALL
            SELECT
                trial_id,
                tstzrange(start_time, end_time) AS range
            FROM
                validations
            UNION ALL
            SELECT
                trial_id,
                tstzrange(start_time, end_time) AS range
            FROM
                checkpoints
        ) AS all_workloads,
        const
    WHERE
        -- `&&` determines whether the ranges overlap.
        const.period && all_workloads.range
),
user_agg AS (
    SELECT
        'username' AS aggregation_type,
        users.username AS aggregation_key,
        sum(workloads.seconds) AS seconds
    FROM
        workloads,
        trials,
        experiments,
        users
    WHERE
        workloads.trial_id = trials.id
        AND trials.experiment_id = experiments.id
        AND experiments.owner_id = users.id
    GROUP BY
        users.username
),
label_agg AS (
    SELECT
        'experiment_label' AS aggregation_type,
        -- This seems to be the most convenient way to convert from a JSONB string value to a normal
        -- string value.
        labels.label #>> '{}' AS aggregation_key,
        sum(workloads.seconds) AS seconds
    FROM
        workloads,
        trials,
        -- An exploded view of experiment labels (one row for each label for each experiment).
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
        workloads.trial_id = trials.id
        AND trials.experiment_id = labels.id
    GROUP BY
        labels.label
),
pool_agg AS (
    SELECT
        'resource_pool' AS aggregation_type,
        experiments.aggregation_key,
        sum(workloads.seconds) AS seconds
    FROM
        workloads,
        trials,
        (
            SELECT
                id,
                coalesce(config #>> '{resources, resource_pool}', 'default') AS aggregation_key
            FROM
                experiments
        ) experiments
    WHERE
        workloads.trial_id = trials.id
        AND trials.experiment_id = experiments.id
    GROUP BY
        experiments.aggregation_key
),
agent_label_agg AS (
    SELECT
        'agent_label' AS aggregation_type,
        experiments.aggregation_key,
        sum(workloads.seconds) AS seconds
    FROM
        workloads,
        trials,
        (
            SELECT
                id,
                coalesce(config #>> '{resources, agent_label}', '') AS aggregation_key
            FROM
                experiments
        ) experiments
    WHERE
        workloads.trial_id = trials.id
        AND trials.experiment_id = experiments.id
    GROUP BY
        experiments.aggregation_key
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
        *
    FROM
        agent_label_agg
    UNION ALL
    SELECT
        'total' AS aggregation_type,
        'total' AS aggregation_key,
        coalesce(sum(workloads.seconds), 0) AS seconds
    FROM
        workloads
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
