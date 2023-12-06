ALTER TABLE run_id_task_id RENAME COLUMN run_id TO trial_id;
ALTER TABLE run_id_task_id RENAME TO trial_id_task_id;

-- DELETE dupes (both records in dupe case).
DELETE FROM trial_id_task_id
WHERE task_id IN (
    SELECT task_id
    FROM trial_id_task_id
    GROUP BY task_id
    HAVING COUNT(*) > 1
);

ALTER TABLE trial_id_task_id ADD CONSTRAINT trial_id_task_id_task_id_key UNIQUE (task_id);

ALTER TABLE checkpoints_v2 DROP CONSTRAINT checkpoints_v2_task_id_fkey;
ALTER TABLE checkpoints_v2
ADD CONSTRAINT checkpoints_v2_task_id_fkey FOREIGN KEY (task_id) REFERENCES trial_id_task_id(task_id)
ON DELETE CASCADE;
