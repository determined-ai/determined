-- determined> \d raw_validations;
-- +----------------+--------------------------+-------------------------------------------------------+
-- | Column         | Type                     | Modifiers                                             |
-- |----------------+--------------------------+-------------------------------------------------------|
-- | id             | integer                  |  not null default nextval('metrics_id_seq'::regclass) |
-- | trial_id       | integer                  |  not null                                             |
-- | end_time       | timestamp with time zone |                                                       |
-- | metrics        | jsonb                    |                                                       |
-- | total_batches  | integer                  |  not null default 0                                   |
-- | trial_run_id   | integer                  |  not null default 0                                   |
-- | archived       | boolean                  |  not null default false                               |
-- | partition_type | metric_partition_type    |  not null default 'VALIDATION'::metric_partition_type |
-- | custom_type    | text                     |                                                       |
-- +----------------+--------------------------+-------------------------------------------------------+
UPDATE raw_validations SET custom_type = 'validation' WHERE custom_type IS NULL;
UPDATE raw_steps SET custom_type = 'training' WHERE custom_type IS NULL;
UPDATE generic_metrics SET custom_type = 'generic' WHERE custom_type IS NULL;

ALTER TABLE raw_validations
ALTER COLUMN custom_type SET DEFAULT 'validation',
ALTER COLUMN custom_type SET NOT NULL;

ALTER TABLE raw_steps
ALTER COLUMN custom_type SET DEFAULT 'training',
ALTER COLUMN custom_type SET NOT NULL;

ALTER TABLE generic_metrics
ALTER COLUMN custom_type SET NOT NULL;

ALTER TABLE metrics
ALTER COLUMN custom_type SET NOT NULL;
