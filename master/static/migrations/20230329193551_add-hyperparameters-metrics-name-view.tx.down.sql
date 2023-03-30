ALTER TABLE projects DROP COLUMN hyperparameters;
DROP TABLE exp_metrics_name;

DROP TRIGGER autoupdate_exp_validation_metrics_name ON raw_validations;

DROP FUNCTION autoupdate_exp_validation_metrics_name;
