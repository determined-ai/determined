/* 
The following tables:

    * raw_steps
    * raw_validations
    * experiment_snapshots
    * checkpoints_v2
    * trials 
    
all contained foreign key constraints from prior migrations. 
In order to remove `ON DELETE CASCADE` from a previously existing constraint, we remove the constraint from the table
and add it back without the cascade during its recreation.
*/

ALTER TABLE raw_steps
DROP CONSTRAINT steps_trial_id_fkey;

ALTER TABLE raw_steps
ADD CONSTRAINT steps_trial_id_fkey FOREIGN KEY (trial_id) REFERENCES trials(id);

ALTER TABLE raw_validations
DROP CONSTRAINT raw_validations_trial_id_fkey;

ALTER TABLE raw_validations
ADD CONSTRAINT raw_validations_trial_id_fkey FOREIGN KEY (trial_id) REFERENCES trials(id);

ALTER TABLE experiment_snapshots 
DROP CONSTRAINT fk_experiment_snapshots_experiments_experiment_id;

ALTER TABLE experiment_snapshots
ADD CONSTRAINT fk_experiment_snapshots_experiments_experiment_id FOREIGN KEY (experiment_id) REFERENCES experiments(id);

ALTER TABLE trials
DROP CONSTRAINT trials_experiment_id_fkey;

ALTER TABLE trials
ADD CONSTRAINT trials_experiment_id_fkey FOREIGN KEY (experiment_id) REFERENCES experiments(id);

ALTER TABLE checkpoints_v2
DROP CONSTRAINT checkpoints_v2_task_id_fkey;

ALTER TABLE checkpoints_v2
ADD CONSTRAINT checkpoints_v2_task_id_fkey FOREIGN KEY (task_id) REFERENCES trial_id_task_id(task_id);
