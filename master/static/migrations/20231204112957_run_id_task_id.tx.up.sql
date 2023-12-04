ALTER TABLE trial_id_task_id RENAME TO run_id_task_id;
ALTER TABLE run_id_task_id RENAME COLUMN trial_id TO run_id;


-- TODO don't drop this constraint. This will be in the many to many.
-- We want to land checkpoint migration first over this.
ALTER TABLE checkpoints_v2 DROP CONSTRAINT checkpoints_v2_task_id_fkey;


ALTER TABLE run_id_task_id DROP CONSTRAINT trial_id_task_id_task_id_key;




-- DROP INDEX trial_id_task_id_task_id_key;
--ALTER TABLE run_id_task_id DROP CONSTRAINT trial_id_task_id_task_id_key;
