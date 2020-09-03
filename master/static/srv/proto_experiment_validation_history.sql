
SELECT (WITH const AS (
        SELECT config->'searcher'->>'metric' AS metric_name,
            (SELECT
                CASE
                    WHEN coalesce((config->'searcher'
                                            ->>'smaller_is_better')::boolean, true)
                    THEN 1
                    ELSE -1
                END) AS sign
        FROM experiments WHERE id = e.id
        ), vals AS (
            SELECT v.trial_id, v.end_time, v.state,
                    (v.metrics->'validation_metrics'->>(const.metric_name))::float8
                    AS searcher_metric
            FROM validations v, trials t, const
            WHERE v.trial_id = t.id and t.experiment_id = e.id and v.state = 'COMPLETED'
        )
        SELECT coalesce(jsonb_agg(v), '[]'::jsonb)
        FROM (
            SELECT n.trial_id, n.end_time, n.searcher_metric
            FROM (
                SELECT v.trial_id, v.end_time, v.searcher_metric,
                    min(const.sign * v.searcher_metric)
                        OVER (ORDER BY v.end_time ASC
                            ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING)
                        AS prev_min_error
                FROM vals v,
                    trials t,
                    const
                WHERE v.trial_id = t.id
                AND v.state = 'COMPLETED'
                AND t.experiment_id = e.id
            ) n, const
            WHERE const.sign * n.searcher_metric < n.prev_min_error
                OR n.prev_min_error IS NULL
            ORDER BY n.end_time asc
        ) v) as validation_history
FROM experiments e
WHERE e.id = $1
