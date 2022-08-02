ALTER TABLE workspaces DROP COLUMN state;
ALTER TABLE projects DROP COLUMN state;
DROP TYPE workspace_state;
