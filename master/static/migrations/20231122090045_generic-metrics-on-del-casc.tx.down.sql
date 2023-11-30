ALTER TABLE generic_metrics
DROP CONSTRAINT generic_metrics_trial_id_fkey;

ALTER TABLE generic_metrics
ADD CONSTRAINT generic_metrics_trial_id_fkey FOREIGN KEY (trial_id) REFERENCES trials(id);
