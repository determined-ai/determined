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

  (SELECT seconds
    FROM resource_aggregates
    WHERE date = ds.period_start AND aggregation_type = 'total'
    LIMIT 1
  ) AS seconds,

  (SELECT jsonb_object_agg(aggregation_key, seconds)
    FROM resource_aggregates
    WHERE date = ds.period_start AND aggregation_type = 'username'
  ) AS by_username,

  (SELECT jsonb_object_agg(aggregation_key, seconds)
    FROM resource_aggregates
    WHERE date = ds.period_start AND aggregation_type = 'experiment_label'
  ) AS by_experiment_label,

  (SELECT jsonb_object_agg(aggregation_key, seconds)
    FROM resource_aggregates
    WHERE date = ds.period_start AND aggregation_type = 'resource_pool'
  ) AS by_resource_pool
FROM
  date_series ds
WHERE
  ds.period_start IN (SELECT DISTINCT date FROM resource_aggregates)
ORDER BY
  ds.period_start
