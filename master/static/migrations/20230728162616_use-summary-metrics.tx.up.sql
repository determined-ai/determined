CREATE OR REPLACE FUNCTION autoupdate_exp_best_trial_metrics() RETURNS trigger AS $$
BEGIN
    WITH bt AS (SELECT id, best_validation_id FROM trials WHERE experiment_id = NEW.experiment_id ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE experiments SET best_trial_id = bt.id, 
    validation_metrics = 
    (
        WITH metrics AS (
            SELECT summary_metrics->'validation_metrics' AS jsonb, searcher_metric_value_signed=searcher_metric_value AS sign FROM trials WHERE id = bt.id
        ), 
        metrics_values AS (
            SELECT jsonb_object_keys(jsonb) AS metrics_key, sign, jsonb -> jsonb_object_keys(jsonb) ->> 'min' AS min, jsonb -> jsonb_object_keys(jsonb) ->> 'max' AS max, jsonb -> jsonb_object_keys(jsonb) ->> 'type' AS type FROM metrics),
        result AS (
        SELECT metrics_key, CASE sign when true then min else max end AS metrics_value, type FROM metrics_values WHERE type = 'number'
        ) SELECT json_object_agg(metrics_key, metrics_value::float8) FROM result
    ) FROM bt
    WHERE experiments.id = NEW.experiment_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER autoupdate_exp_validation_metrics_name ON raw_validations;

DROP FUNCTION autoupdate_exp_validation_metrics_name;

DROP TABLE exp_metrics_name;