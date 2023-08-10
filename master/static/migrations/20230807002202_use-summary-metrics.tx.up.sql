DROP TRIGGER autoupdate_exp_best_trial_metrics ON trials;

DROP FUNCTION autoupdate_exp_best_trial_metrics;

DROP TRIGGER autoupdate_exp_validation_metrics_name ON raw_validations;

DROP FUNCTION autoupdate_exp_validation_metrics_name;

DROP TABLE exp_metrics_name;

ALTER TABLE experiments DROP COLUMN validation_metrics;