DROP TABLE experiment_groups;
ALTER TABLE experiments DROP COLUMN group_id;

DROP INDEX IF EXISTS ix_experiment_groups_id;