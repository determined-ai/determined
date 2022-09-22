DROP TABLE project_experiment_groups;
ALTER TABLE experiments DROP COLUMN group_id;

DROP INDEX IF EXISTS ix_project_experiment_groups_id;