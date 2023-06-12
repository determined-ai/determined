ALTER TABLE tasks ADD COLUMN trial_id INTEGER NULL;

INSERT INTO tasks (task_id, task_type, start_time, end_time, job_id, log_version, trial_id)
SELECT
    'backfilled-' || t.id,
    'TRIAL',
    t.start_time,
    t.end_time,
    e.job_id,
    0, -- They must be very, very old. Prior to log_version 1.
    t.id
FROM trials t
JOIN experiments e ON t.experiment_id = e.id
WHERE t.task_id IS NULL;

UPDATE trials t
SET task_id = tk.task_id
FROM tasks tk
WHERE tk.trial_id = t.id;

ALTER TABLE tasks DROP COLUMN trial_id;

ALTER TABLE trials ALTER COLUMN task_id SET NOT NULL;
