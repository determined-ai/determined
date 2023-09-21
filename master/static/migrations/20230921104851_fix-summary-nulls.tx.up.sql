WITH summary_metrics_with_mean AS (
  SELECT
    trials.id,
    jsonb_object_agg(
      metric_group,
      CASE
        WHEN summary_metrics->metric_group = 'null'::JSONB OR
          summary_metrics->metric_group IS NULL THEN '{}'::JSONB
        ELSE summary_metrics->metric_group
      END
    ) AS summary_metrics
  FROM
    trials,
    jsonb_object_keys(summary_metrics) as metric_group
  WHERE
    summary_metrics IS NOT NULL
  GROUP BY
    trials.id
)
UPDATE
  trials
SET
  summary_metrics = summary_metrics_with_mean.summary_metrics
FROM
  summary_metrics_with_mean
WHERE
  trials.id = summary_metrics_with_mean.id;
