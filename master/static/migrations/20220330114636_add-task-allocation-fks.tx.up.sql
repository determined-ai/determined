-- NULLify entries that would cause an integrity error.
UPDATE trials SET task_id = NULL WHERE task_id NOT IN (SELECT task_id FROM tasks);

ALTER TABLE trials
ADD CONSTRAINT task_id_fkey
FOREIGN KEY (task_id) REFERENCES tasks (task_id);

-- Since we clear out allocation_sessions on master init anyway,
-- clear them out here to avoid conflicts.
DELETE FROM allocation_sessions;

ALTER TABLE allocation_sessions
ADD CONSTRAINT allocation_id_fkey
FOREIGN KEY (allocation_id) REFERENCES allocations (allocation_id);
