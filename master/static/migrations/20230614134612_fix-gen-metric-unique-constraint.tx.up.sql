-- determined> \d generic_metrics;
-- +----------------+--------------------------+-------------------------------------------------------+
-- | Column         | Type                     | Modifiers                                             |
-- |----------------+--------------------------+-------------------------------------------------------|
-- | trial_id       | integer                  |  not null                                             |
-- | end_time       | timestamp with time zone |                                                       |
-- | metrics        | jsonb                    |                                                       |
-- | total_batches  | integer                  |  not null default 0                                   |
-- | trial_run_id   | integer                  |  not null default 0                                   |
-- | archived       | boolean                  |  not null default false                               |
-- | id             | integer                  |  not null default nextval('metrics_id_seq'::regclass) |
-- | partition_type | metric_partition_type    |  not null default 'GENERIC'::metric_partition_type    |
-- | custom_type    | text                     |  not null                                             |
-- +----------------+--------------------------+-------------------------------------------------------+
-- Indexes:
--     "generic_metrics_trial_id_total_batches_run_id_unique" UNIQUE, btree (trial_id, total_batches, trial_run_id)
--     "generic_metrics_archived" btree (archived)
-- Check constraints:
--     "custom_type_check" CHECK (custom_type <> ALL (ARRAY['validation'::text, 'training'::text, 'avg_metrics'::text, 'validation_metrics'::text]))
-- Foreign-key constraints:
--     "generic_metrics_trial_id_fkey" FOREIGN KEY (trial_id) REFERENCES trials(id)
-- Partition of: public.metrics FOR VALUES IN ('GENERIC')
-- Partition constraint: ((partition_type IS NOT NULL) AND ((partition_type)::anyenum OPERATOR(pg_catalog.=) ANY (ARRAY['GENERIC'::metric_partition_type])))

DROP INDEX generic_metrics_trial_id_total_batches_run_id_unique;
CREATE UNIQUE INDEX generic_metrics_trial_id_total_batches_run_id_custom_type_unique ON generic_metrics (trial_id, total_batches, trial_run_id, custom_type);
