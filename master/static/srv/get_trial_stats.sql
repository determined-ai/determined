WITH const AS (
    SELECT config->'searcher'->>'metric' AS metric_name,
        (
            SELECT CASE
                    WHEN coalesce(
                        (config->'searcher'->>'smaller_is_better')::boolean,
                        true
                    ) THEN 1
                    ELSE -1
                END
        ) as sign
    FROM experiments
    WHERE id = $1 -- $experimentid
)
SELECT row_to_json(x)
FROM (
        SELECT const.metric_name,
            (
                SELECT *
                FROM (
                        SELECT step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                        FROM (
                                SELECT (
                                        SELECT row_to_json(s)
                                        FROM (
                                                SELECT s.end_time,
                                                    s.id,
                                                    s.start_time,
                                                    s.state,
                                                    s.trial_id,
                                                    s.num_batches,
                                                    s.prior_batches_processed,
                                                    (
                                                        SELECT row_to_json(v)
                                                        FROM (
                                                                SELECT v.end_time,
                                                                    v.id,
                                                                    v.metrics,
                                                                    v.start_time,
                                                                    v.state,
                                                                    v.step_id,
                                                                    v.trial_id
                                                                FROM validations v
                                                                WHERE v.trial_id = s.trial_id
                                                                    AND v.step_id = s.id
                                                            ) v
                                                    ) AS validation
                                                FROM steps s
                                                WHERE s.id = c.step_id
                                                    AND s.trial_id = c.trial_id
                                            ) s
                                    ) AS step
                                FROM checkpoints c,
                                    trials t,
                                    const
                                WHERE c.trial_id = $2
                                    AND t.id = $2
                            ) val_step
                        WHERE (
                                (
                                    step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                                )::float8 IS NOT NULL
                            )
                        ORDER BY (
                                (step->>'id')::int
                            ) DESC
                        LIMIT 1
                    ) v
            ) AS latest_validation,
            (
                SELECT *
                FROM (
                        SELECT step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                        FROM (
                                SELECT (
                                        SELECT row_to_json(s)
                                        FROM (
                                                SELECT s.end_time,
                                                    s.id,
                                                    s.start_time,
                                                    s.state,
                                                    s.trial_id,
                                                    s.num_batches,
                                                    s.prior_batches_processed,
                                                    (
                                                        SELECT row_to_json(v)
                                                        FROM (
                                                                SELECT v.end_time,
                                                                    v.id,
                                                                    v.metrics,
                                                                    v.start_time,
                                                                    v.state,
                                                                    v.step_id,
                                                                    v.trial_id
                                                                FROM validations v
                                                                WHERE v.trial_id = s.trial_id
                                                                    AND v.step_id = s.id
                                                            ) v
                                                    ) AS validation
                                                FROM steps s
                                                WHERE s.id = c.step_id
                                                    AND s.trial_id = c.trial_id
                                            ) s
                                    ) AS step
                                FROM checkpoints c,
                                    trials t,
                                    const
                                WHERE c.trial_id = $2
                                    AND t.id = $2 -- $1
                            ) val_step
                        WHERE (
                                (
                                    step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                                )::float8 IS NOT NULL
                            )
                        ORDER BY (
                                const.sign * (
                                    step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                                )::float8
                            ) ASC
                        LIMIT 1
                    ) v
            ) AS best_validation,
            (
                SELECT *
                FROM (
                        SELECT cp_step.uuid AS uuid
                        FROM (
                                SELECT c.id,
                                    c.trial_id,
                                    c.step_id,
                                    c.state,
                                    c.start_time,
                                    c.end_time,
                                    c.uuid,
                                    c.resources,
                                    c.metadata,
                                    (
                                        SELECT row_to_json(s)
                                        FROM (
                                                SELECT s.end_time,
                                                    s.id,
                                                    s.start_time,
                                                    s.state,
                                                    s.trial_id,
                                                    s.num_batches,
                                                    s.prior_batches_processed,
                                                    (
                                                        SELECT row_to_json(v)
                                                        FROM (
                                                                SELECT v.end_time,
                                                                    v.id,
                                                                    v.metrics,
                                                                    v.start_time,
                                                                    v.state,
                                                                    v.step_id,
                                                                    v.trial_id
                                                                FROM validations v
                                                                WHERE v.trial_id = s.trial_id
                                                                    AND v.step_id = s.id
                                                            ) v
                                                    ) AS validation
                                                FROM steps s
                                                WHERE s.id = c.step_id
                                                    AND s.trial_id = c.trial_id
                                            ) s
                                    ) AS step
                                FROM checkpoints c,
                                    trials t,
                                    const
                                WHERE c.trial_id = $2
                                    AND t.id = $2 -- $1
                            ) cp_step
                        WHERE (
                                (
                                    step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                                )::float8 IS NOT NULL
                            )
                        ORDER BY (
                                const.sign * (
                                    step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                                )::float8
                            ) ASC
                        LIMIT 1
                    ) c
            ) AS best_checkpoint
        FROM const
    ) x
