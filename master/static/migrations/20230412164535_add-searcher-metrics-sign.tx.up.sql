ALTER TABLE trials ADD COLUMN searcher_metric_value_signed float8 DEFAULT NULL;
CREATE INDEX ix_trials_metric_value ON trials USING btree (searcher_metric_value_signed);

ALTER TABLE experiments ADD COLUMN best_trial_id int DEFAULT NULL;

ALTER TABLE experiments ADD COLUMN validation_metrics JSONB NULL;
CREATE INDEX ix_experiments_validation_metrics ON experiments USING GIN (validation_metrics);

WITH si AS (
SELECT id,
CASE
WHEN coalesce((ex.config->'searcher'->>'smaller_is_better')::boolean, true)
    THEN 1
    ELSE -1.0 
END AS sign
FROM experiments ex)
UPDATE trials SET searcher_metric_value_signed = si.sign * searcher_metric_value FROM si WHERE experiment_id = si.id;

WITH sv AS (
SELECT DISTINCT ON (experiment_id) experiment_id, id, searcher_metric_value_signed FROM trials WHERE searcher_metric_value_signed IS NOT NULL GROUP BY experiment_id, id ORDER BY experiment_id, searcher_metric_value_signed)
UPDATE experiments SET best_trial_id = sv.id FROM sv WHERE experiments.id = sv.experiment_id;

WITH vm AS (
    SELECT e.id, metrics -> 'validation_metrics' AS validations_metrics FROM experiments e, trials t, validations v WHERE e.best_trial_id = t.id AND t.best_validation_id = v.id
) UPDATE experiments SET validation_metrics = vm.validations_metrics FROM vm WHERE experiments.id = vm.id;

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
CREATE TRIGGER autoupdate_exp_best_trial_metrics
AFTER UPDATE OF best_validation_id ON trials
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_best_trial_metrics();

