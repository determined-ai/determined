ALTER TABLE raw_validations
ADD CONSTRAINT custom_type_check
CHECK (custom_type = 'validation');

ALTER TABLE raw_steps
ADD CONSTRAINT custom_type_check
CHECK (custom_type = 'training');
