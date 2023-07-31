CREATE OR REPLACE FUNCTION autoupdate_exp_best_trial_metrics() RETURNS trigger AS $$
BEGIN
    WITH bt AS (SELECT id, best_validation_id FROM trials WHERE experiment_id = NEW.experiment_id ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE experiments SET best_trial_id = bt.id, 
    validation_metrics = 
    (SELECT metrics -> 'validation_metrics' FROM validations v WHERE v.id = bt.best_validation_id) FROM bt
    WHERE experiments.id = NEW.experiment_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


DROP TABLE exp_metrics_name;