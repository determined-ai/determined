CREATE OR REPLACE FUNCTION autoupdate_exp_best_trial_metrics_on_delete() RETURNS trigger AS $$
BEGIN
    WITH bt AS (
        SELECT id, best_validation_id 
        FROM trials 
        WHERE experiment_id = OLD.experiment_id 
        ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE experiments SET best_trial_id = bt.id FROM bt
    WHERE experiments.id = OLD.experiment_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_exp_best_trial_metrics_on_run_delete
AFTER DELETE ON runs
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_best_trial_metrics_on_delete();
