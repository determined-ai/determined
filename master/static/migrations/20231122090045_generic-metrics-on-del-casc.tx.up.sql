/* 
generic_metrics needs delete cascade added to its foreign key constraint to trials(id) since 
trials(experiment_id) has delete cascade constraint to experiments(id).
*/

ALTER TABLE generic_metrics
DROP CONSTRAINT generic_metrics_trial_id_fkey;

ALTER TABLE generic_metrics
ADD CONSTRAINT generic_metrics_trial_id_fkey FOREIGN KEY (trial_id) REFERENCES trials(id)
ON DELETE CASCADE;
