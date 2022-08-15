ALTER TABLE workspaces DROP COLUMN state;
ALTER TABLE workspaces DROP COLUMN error_message;
ALTER TABLE projects DROP COLUMN state;
ALTER TABLE projects DROP COLUMN error_message;
DROP TYPE workspace_state;
