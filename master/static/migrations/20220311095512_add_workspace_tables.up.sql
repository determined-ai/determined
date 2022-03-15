-- Tables
CREATE TABLE workspaces (
  id SERIAL PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  archived BOOLEAN,
  created_at timestamp with time zone NOT NULL DEFAULT NOW(),
<<<<<<< HEAD
  user_id INT REFERENCES users(id),
  immutable BOOLEAN DEFAULT FALSE
=======
  user_id INT REFERENCES users(id)
>>>>>>> 12a767bd4 (indexes and default uncategorized obj)
);
CREATE TABLE projects (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  archived BOOLEAN,
  created_at timestamp with time zone NOT NULL DEFAULT NOW(),
  notes JSONB,
  workspace_id INT REFERENCES workspaces(id),
<<<<<<< HEAD
  user_id INT REFERENCES users(id),
  immutable BOOLEAN DEFAULT FALSE,
  UNIQUE (name, workspace_id)
=======
  user_id INT REFERENCES users(id)
>>>>>>> 12a767bd4 (indexes and default uncategorized obj)
);
CREATE TABLE project_models (
  name TEXT,
  checkpoint_id INT,
  created_at timestamp with time zone NOT NULL DEFAULT NOW(),
  project_id INT REFERENCES projects(id)
);
ALTER TABLE experiments ADD COLUMN project_id INT REFERENCES projects(id) NULL;

-- Indexes
CREATE INDEX ix_experiments_project_id ON public.experiments USING btree (project_id);
CREATE INDEX ix_projects_workspace_id ON public.projects USING btree (workspace_id);
CREATE INDEX ix_projects_user_id ON public.projects USING btree (user_id);
<<<<<<< HEAD
CREATE INDEX ix_project_models_project_id ON public.project_models USING btree (project_id);
=======
>>>>>>> 12a767bd4 (indexes and default uncategorized obj)
CREATE INDEX ix_workspaces_user_id ON public.workspaces USING btree (user_id);

-- Default workspace and project objects
WITH admin AS (
  SELECT id FROM users WHERE username = 'admin' LIMIT 1
),
worker AS (
<<<<<<< HEAD
  INSERT INTO workspaces (name, archived, user_id, immutable)
  SELECT 'Uncategorized', false, admin.id, true
=======
  INSERT INTO workspaces (name, archived, user_id)
  SELECT 'Uncategorized', false, admin.id
>>>>>>> 12a767bd4 (indexes and default uncategorized obj)
  FROM admin
  RETURNING workspaces.id
),
newp AS (
<<<<<<< HEAD
  INSERT INTO projects (name, description, archived, workspace_id, user_id, immutable)
  SELECT 'Uncategorized', '', false, worker.id, admin.id, true
=======
  INSERT INTO projects (name, description, archived, workspace_id, user_id)
  SELECT 'Uncategorized', '', false, worker.id, admin.id
>>>>>>> 12a767bd4 (indexes and default uncategorized obj)
  FROM admin, worker
  RETURNING projects.id
)
UPDATE experiments SET project_id = (SELECT id FROM newp);
