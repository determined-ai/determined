DROP TABLE projects CASCADE;
DROP TABLE workspaces;
DROP TABLE project_models;
ALTER TABLE experiments DROP COLUMN project_id;

DROP INDEX IF EXISTS ix_experiments_project_id;
DROP INDEX IF EXISTS ix_projects_workspace_id;
DROP INDEX IF EXISTS ix_projects_user_id;
DROP INDEX IF EXISTS ix_project_models_project_id;
DROP INDEX IF EXISTS ix_workspaces_user_id;
