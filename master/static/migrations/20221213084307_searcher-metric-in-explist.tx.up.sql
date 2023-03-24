
ALTER TABLE trials ADD column searcher_metric_value float8 DEFAULT NULL;

UPDATE trials t SET
  searcher_metric_value =
    (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8
FROM
  validations v,
  experiments e
WHERE
  t.best_validation_id = v.id
  AND e.id = t.experiment_id;
