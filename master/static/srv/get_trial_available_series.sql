SELECT json_agg(q.labels) AS labels FROM (
    SELECT DISTINCT m.labels
    FROM trial_profiler_metrics m
    WHERE m.labels @> $1::jsonb
) q;
