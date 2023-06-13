ALTER TABLE raw_validations
ADD CONSTRAINT custom_type_check
CHECK (custom_type = 'validation');

ALTER TABLE raw_steps
ADD CONSTRAINT custom_type_check
CHECK (custom_type = 'training');

-- Post PG v10 we want to use custom_type as the partition key.
DELETE FROM generic_metrics WHERE custom_type IN ('validation', 'training', 'avg_metrics', 'validation_metrics');

ALTER TABLE generic_metrics
ADD CONSTRAINT custom_type_check
CHECK (custom_type NOT IN ('validation', 'training', 'avg_metrics', 'validation_metrics'));
