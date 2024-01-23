WITH date_series AS (
  SELECT generate_series(
    GREATEST($1::date, (SELECT min(date) FROM resource_aggregates)),
    LEAST($2::date, (SELECT max(date) FROM resource_aggregates)),
    '1 day'::interval
  ) AS period_start
)
SELECT
  to_char(ds.period_start, 'YYYY-MM-DD') AS period_start,
  'RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY' AS period,

  -- We divide by count since we have multiple rows with the same total.
  (SUM(ra_total.seconds) FILTER (WHERE ra_total.aggregation_key IS NOT NULL) /
    COUNT(*) FILTER (WHERE ra_total.aggregation_key IS NOT NULL)) AS seconds,

  jsonb_object_agg(
    ra_username.aggregation_key, ra_username.seconds
  ) FILTER (WHERE ra_username.aggregation_key IS NOT NULL) AS by_username,

  jsonb_object_agg(
    ra_experiment_label.aggregation_key, ra_experiment_label.seconds
  ) FILTER (WHERE ra_experiment_label.aggregation_key IS NOT NULL) AS by_experiment_label,

  jsonb_object_agg(
    ra_resource_pool.aggregation_key, ra_resource_pool.seconds
  ) FILTER (WHERE ra_resource_pool.aggregation_key IS NOT NULL) AS by_resource_pool
FROM
  date_series ds
LEFT JOIN
  resource_aggregates ra_username ON ds.period_start = ra_username.date AND
    ra_username.aggregation_type = 'username'
LEFT JOIN
  resource_aggregates ra_resource_pool ON ds.period_start = ra_resource_pool.date AND
    ra_resource_pool.aggregation_type = 'resource_pool'
LEFT JOIN
  resource_aggregates ra_experiment_label ON ds.period_start = ra_experiment_label.date AND
    ra_experiment_label.aggregation_type = 'experiment_label'
LEFT JOIN
  resource_aggregates ra_total ON ds.period_start = ra_total.date AND
    ra_total.aggregation_type = 'total'
WHERE
  ds.period_start IN (SELECT DISTINCT date FROM resource_aggregates)
GROUP BY
  ds.period_start
ORDER BY
  ds.period_start
