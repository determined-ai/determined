DROP INDEX validations_trial_id_total_batches_run_id_unique;

DROP INDEX validations_archived;

ALTER TABLE raw_validations ADD CONSTRAINT validations_trial_id_run_id_total_batches_unique UNIQUE (trial_id, trial_run_id, total_batches);

DROP VIEW public.trials_augmented_view;

DROP AGGREGATE jsonb_collect(jsonb);

DROP TABLE trials_collections;

DROP INDEX trials_tags_index;

ALTER TABLE trials DROP column IF EXISTS tags;
