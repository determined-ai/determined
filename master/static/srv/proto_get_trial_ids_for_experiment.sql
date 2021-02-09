WITH searcher_info AS (
    SELECT config->'searcher'->>'metric' AS metric_name,
           (
               SELECT CASE
                          WHEN coalesce(
                                  (
                                              config->'searcher'->>'smaller_is_better'
                                      )::boolean,
                                  true
                              ) THEN 1
                          ELSE -1
                          END
           ) AS sign
    FROM experiments e
    WHERE e.id = $1
), filtered_experiment_trials AS (
    SELECT
        t.id AS id,
        'STATE_' || t.state AS state,
        t.start_time,
        t.end_time,
        coalesce(t.end_time, now()) - t.start_time AS duration,
        (
            SELECT s.total_batches
            FROM steps s
            WHERE s.trial_id = t.id
                AND s.state = 'COMPLETED'
            ORDER BY s.id DESC
            LIMIT 1
        ) AS total_batches_processed,
        (
           CASE WHEN t.best_validation_id IS NOT NULL THEN
                (SELECT searcher_info.sign * (v.metrics->'validation_metrics'->>searcher_info.metric_name)::float8
                FROM validations v
                WHERE v.id = t.best_validation_id
                LIMIT 1)
            ELSE
                -- For trials before `public.trials.best_validation_id` was added.
                (SELECT searcher_info.sign * (v.metrics->'validation_metrics'->>searcher_info.metric_name)::float8
                 FROM validations v
                 WHERE v.trial_id = t.id
                 ORDER BY searcher_info.sign * (v.metrics->'validation_metrics'->>searcher_info.metric_name)::float8 ASC
                 LIMIT 1)
            END
        ) as best_signed_search_metric,
        (
           SELECT searcher_info.sign * (v.metrics->'validation_metrics'->>searcher_info.metric_name)::float8
           FROM validations v
           WHERE v.trial_id = t.id
             AND v.state = 'COMPLETED'
           ORDER BY v.id DESC
           LIMIT 1
        ) as latest_signed_search_metric
    FROM trials t, searcher_info
    WHERE t.experiment_id = $1
      AND ($2 = '' OR t.state IN (SELECT unnest(string_to_array($2, ','))::trial_state))
), page_info AS (
    SELECT public.page_info((SELECT COUNT(*) AS count FROM filtered_experiment_trials), $3, $4) AS page_info
)
SELECT
    (SELECT coalesce(json_agg(paginated_experiment_trials), '[]'::json) FROM (
        SELECT id FROM filtered_experiment_trials
        ORDER BY %s
        OFFSET (SELECT p.page_info->>'start_index' FROM page_info p)::bigint
        LIMIT (SELECT (p.page_info->>'end_index')::bigint - (p.page_info->>'start_index')::bigint FROM page_info p)
    ) AS paginated_experiment_trials) AS trials,
    (SELECT p.page_info FROM page_info p) AS pagination
