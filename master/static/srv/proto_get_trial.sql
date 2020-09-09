SELECT t.id,
    t.experiment_id,
    'STATE_' || t.state AS state,
    t.start_time,
    t.end_time,
    t.hparams,
    (
        SELECT s.prior_batches_processed + s.num_batches
        FROM steps s
        WHERE s.trial_id = t.id
            AND s.state = 'COMPLETED'
        ORDER BY s.id DESC
        LIMIT 1
    ) AS total_batches_processed,
    extras.best_checkpoint,
    extras.best_validation,
    extras.latest_validation
FROM trials t
    JOIN (
        WITH const AS (
            SELECT config->'searcher'->>'metric' AS metric_name,
                (
                    SELECT CASE
                            WHEN coalesce(
                                (
                                    config->'searcher'->>'smaller_is_better'
                                )::boolean,
                                true
                            ) THEN -1 -- so we can order by DESC to get the highest value
                            ELSE 1
                        END
                ) AS sign,
                t.id AS trial_id
            FROM experiments e
                INNER JOIN trials t ON t.experiment_id = e.id
            WHERE t.id = 20
        ),
        w_validations AS (
            SELECT v.trial_id,
                v.step_id,
                v.end_time,
                v.state,
                (
                    v.metrics->'validation_metrics'->>(const.metric_name)
                )::float8 AS searcher_metric,
                (
                    (
                        v.metrics->'validation_metrics'->>(const.metric_name)
                    )::float8 * const.sign
                ) AS searcher_metric_value
            FROM validations v
                JOIN const ON const.trial_id = v.trial_id
            WHERE v.state = 'COMPLETED'
                AND (
                    v.metrics->'validation_metrics'->>(const.metric_name)
                ) IS NOT NULL
        ),
        best_validation AS (
            SELECT v.*
            FROM w_validations v
            ORDER BY v.searcher_metric_value DESC
            LIMIT 1
        ), latest_validation AS (
            SELECT v.*
            FROM w_validations v
            ORDER BY end_time DESC
            LIMIT 1
        ), best_checkpoint AS (
            SELECT c.*
            FROM checkpoints c
                JOIN w_validations v ON c.step_id = v.step_id
            WHERE c.trial_id = 20
            ORDER BY v.searcher_metric_value DESC
            LIMIT 1
        )
        SELECT bv.searcher_metric AS best_validation,
            lv.searcher_metric AS latest_validation,
            bc.id AS best_checkpoint,
            t.id
        FROM trials t
            INNER JOIN best_validation bv ON bv.trial_id = t.id
            INNER JOIN latest_validation lv ON lv.trial_id = t.id
            INNER JOIN best_checkpoint bc ON bc.trial_id = t.id
        LIMIT 1
    ) extras ON t.id = extras.id
WHERE t.id = $1
