CREATE TYPE workspace_state AS ENUM ('UNSPECIFIED', 'DELETED', 'DELETING', 'DELETE_FAILED');
ALTER TABLE workspaces ADD COLUMN state workspace_state DEFAULT 'UNSPECIFIED';
ALTER TABLE workspaces ADD COLUMN error_message TEXT;
ALTER TABLE projects ADD COLUMN state workspace_state DEFAULT 'UNSPECIFIED';
ALTER TABLE projects ADD COLUMN error_message TEXT;
