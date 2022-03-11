CREATE TABLE workspaces (
  id SERIAL PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  archived BOOLEAN,
  user_id INT REFERENCES users(id)
);
CREATE TABLE projects (
  id SERIAL PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  description TEXT,
  archived BOOLEAN,
  workspace_id INT REFERENCES workspaces(id),
  user_id INT REFERENCES users(id)
);
CREATE TABLE notes (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  contents TEXT,
  project_id INT REFERENCES projects(id)
);
CREATE TABLE project_models (
  name TEXT,
  checkpoint_id INT
);
ALTER TABLE experiments ADD COLUMN project_id INT REFERENCES projects(id) NULL;
