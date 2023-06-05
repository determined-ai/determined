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
ALTER TABLE raw_validations ALTER COLUMN custom_type SET DEFAULT 'validation';
ALTER TABLE raw_steps ALTER COLUMN custom_type SET DEFAULT 'training';
ALTER TABLE generic_metrics ALTER COLUMN custom_type SET DEFAULT 'generic';
