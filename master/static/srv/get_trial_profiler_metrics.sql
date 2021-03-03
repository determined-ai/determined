SELECT
       m.values AS values,
       m.batches AS batches,
       m.timestamps AS timestamps,
       m.labels AS labels
FROM trial_profiler_metrics m
WHERE m.labels @> $1::jsonb
ORDER by m.id
OFFSET $2 LIMIT $3;