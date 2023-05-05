CREATE FUNCTION public.experiments_best_checkpoints_by_metric(exp experiments, metric text, smaller_is_better boolean, lim integer) RETURNS SETOF checkpoints
    LANGUAGE sql STABLE
    AS $$
    WITH const AS (
        SELECT
            coalesce(smaller_is_better, (exp.config->'searcher'->>'smaller_is_better')::boolean, true) AS smaller_is_better,
            coalesce(metric, exp.config->'searcher'->>'metric') AS metric
    )
    SELECT c.*
    FROM (
        SELECT c.*
        FROM const, trials t, LATERAL best_checkpoint_by_metric(t.id, const.metric, const.smaller_is_better) c WHERE t.experiment_id = exp.id
    ) c, validations v, const WHERE (c.trial_id, c.step_id) = (v.trial_id, v.step_id)
    ORDER BY (SELECT CASE WHEN const.smaller_is_better THEN 1 ELSE -1 END) * (v.metrics->'validation_metrics'->>const.metric)::float8 ASC
    LIMIT lim
$$;
