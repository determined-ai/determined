-- Swap checkpoints v2 constraint over to tasks like it should have been.
ALTER TABLE checkpoints_v2 DROP CONSTRAINT checkpoints_v2_task_id_fkey;
ALTER TABLE checkpoints_v2 ADD CONSTRAINT checkpoints_v2_task_id_fkey
  FOREIGN KEY (task_id) REFERENCES tasks(task_id) ON DELETE CASCADE;

-- Rename trial_id_task_id and trial_id.
ALTER TABLE trial_id_task_id RENAME TO run_id_task_id;
ALTER TABLE run_id_task_id RENAME COLUMN trial_id TO run_id;

-- Drop uniqueness constraint.
ALTER TABLE run_id_task_id DROP CONSTRAINT trial_id_task_id_task_id_key;
