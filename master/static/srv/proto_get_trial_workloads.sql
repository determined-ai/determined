WITH validations_vt AS (
    SELECT
        row_to_json(r1) AS validation,
        total_batches,
        end_time,
        metrics
    FROM (
        SELECT
            v.end_time,
            v.total_batches,
            v.metrics -> 'num_inputs' AS num_inputs,
            jsonb_build_object('avg_metrics', v.metrics -> 'validation_metrics') AS metrics
        FROM
            validations v
        WHERE
            v.trial_id = $1) AS r1
),
trainings_vt AS (
    SELECT
        row_to_json(r1) AS training,
        total_batches,
        end_time,
        metrics
    FROM (
        SELECT
            s.end_time,
            CASE WHEN $5 = TRUE THEN
                s.metrics
            ELSE
                jsonb_build_object('avg_metrics', s.metrics -> 'avg_metrics')
            END AS metrics,
            s.metrics -> 'num_inputs' AS num_inputs,
            s.total_batches
        FROM
            steps s
        WHERE
            s.trial_id = $1
            AND $4 = 'FILTER_OPTION_UNSPECIFIED') AS r1
),
checkpoints_vt AS (
    SELECT
        row_to_json(r1) AS checkpoint,
        total_batches,
        end_time
    FROM (
        SELECT
            'STATE_' || c.state AS state,
            c.report_time AS end_time,
            c.uuid,
            c.steps_completed AS total_batches,
            c.resources,
            c.metadata
        FROM
            checkpoints_view c
        WHERE
            c.trial_id = $1
            AND $4 != 'FILTER_OPTION_VALIDATION') AS r1
),
workloads AS (
    SELECT
        v.validation::jsonb AS validation,
        t.training::jsonb AS training,
        c.checkpoint::jsonb AS checkpoint,
        coalesce(t.total_batches, v.total_batches, c.total_batches) AS total_batches,
        coalesce(t.end_time, v.end_time, c.end_time) AS end_time,
        CASE WHEN $6 = 'METRIC_TYPE_VALIDATION' THEN
            v.metrics
        WHEN $6 = 'METRIC_TYPE_TRAINING' THEN
            t.metrics
        ELSE
            coalesce(t.metrics, v.metrics)
        END AS sort_metrics
    FROM
        trainings_vt t
        FULL JOIN checkpoints_vt c ON FALSE
        FULL JOIN validations_vt v ON FALSE
),
page_info AS (
    SELECT
        public.page_info ((
            SELECT
                COUNT(*) AS count
        FROM workloads), $2::int, $3::int) AS page_info
)
SELECT
    (
        SELECT
            jsonb_agg(w)
        FROM (
            SELECT
                validation,
                training,
                CHECKPOINT
            FROM
                workloads
            ORDER BY
                (% s)::float % s NULLS LAST,
                total_batches % s,
                end_time % s OFFSET (
                    SELECT
                        p.page_info ->> 'start_index'
                    FROM page_info p)::bigint
            LIMIT (
                SELECT
                    (p.page_info ->> 'end_index')::bigint - (p.page_info ->> 'start_index')::bigint
                FROM
                    page_info p)) w) AS workloads,
    (
        SELECT
            p.page_info
        FROM
            page_info p) AS pagination
