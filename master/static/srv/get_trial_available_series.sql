SELECT DISTINCT m.labels
FROM trial_profiler_metrics m
ORDER BY m.id
OFFSET $1 LIMIT $2