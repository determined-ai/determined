/* 
system_metrics needs delete cascade added to its foreign key constraint to runs(id) since 
runs(experiment_id) has delete cascade constraint to experiments(id).
*/

ALTER TABLE system_metrics
DROP CONSTRAINT system_metrics_trial_id_fkey;

ALTER TABLE system_metrics
ADD CONSTRAINT system_metrics_trial_id_fkey FOREIGN KEY (trial_id) REFERENCES runs(id)
ON DELETE CASCADE;
