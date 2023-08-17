CREATE OR REPLACE FUNCTION autoupdate_exp_best_trial_metrics() RETURNS trigger AS $$
BEGIN
    WITH bt AS (
        SELECT id, best_validation_id 
        FROM trials 
        WHERE experiment_id = NEW.experiment_id 
        ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE experiments SET best_trial_id = bt.id FROM bt
    WHERE experiments.id = NEW.experiment_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS autoupdate_exp_validation_metrics_name ON raw_validations;

DROP FUNCTION IF EXISTS autoupdate_exp_validation_metrics_name;

DROP TABLE IF EXISTS exp_metrics_name;

ALTER TABLE experiments DROP COLUMN IF EXISTS validation_metrics;

WITH summary_metrics_with_mean AS (
  SELECT 
    trials.id, 
    jsonb_object_agg(
      metric_group, 
      (
        SELECT 
          jsonb_object_agg(
            m.key, 
            CASE WHEN m.value -> 'type' = '"number"' :: jsonb AND m.value -> 'sum' IS NOT NULL THEN m.value || jsonb_build_object(
              'mean', 
              (m.value ->> 'sum'):: float8 / (m.value ->> 'count'):: int
            ) ELSE m.value END
          ) 
        FROM 
          jsonb_each(summary_metrics -> metric_group) AS m(key, value)
      )
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
