ALTER TABLE trials DROP COLUMN searcher_metric_value_signed;
DROP INDEX ix_trials_metric_value;

ALTER TABLE experiments DROP COLUMN best_trial_id;
ALTER TABLE experiments DROP COLUMN validation_metrics;
DROP INDEX ix_experiments_validation_metrics;

DROP FUNCTION autoupdate_exp_best_trial_metrics;
DROP TRIGGER autoupdate_exp_best_trial_metrics ON trials;