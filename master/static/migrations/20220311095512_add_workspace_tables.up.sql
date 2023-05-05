-- Tables
CREATE TABLE workspaces (
    id serial PRIMARY KEY,
    name text UNIQUE NOT NULL,
    archived boolean,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    user_id int REFERENCES users (id),
    immutable BOOLEAN DEFAULT FALSE
);

CREATE TABLE projects (
    id serial PRIMARY KEY,
    name text NOT NULL,
    description text,
    archived boolean,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    notes jsonb,
    workspace_id int REFERENCES workspaces (id),
    user_id int REFERENCES users (id),
    immutable BOOLEAN DEFAULT FALSE,
    UNIQUE (name, workspace_id)
);

CREATE TABLE project_models (
    name text,
    checkpoint_id int,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    project_id int REFERENCES projects (id)
);

ALTER TABLE experiments
    ADD COLUMN project_id INT REFERENCES projects (id) NULL;

-- Indexes
CREATE INDEX ix_experiments_project_id ON public.experiments USING btree (project_id);

CREATE INDEX ix_projects_workspace_id ON public.projects USING btree (workspace_id);

CREATE INDEX ix_projects_user_id ON public.projects USING btree (user_id);

CREATE INDEX ix_project_models_project_id ON public.project_models USING btree (project_id);

CREATE INDEX ix_workspaces_user_id ON public.workspaces USING btree (user_id);

-- Default workspace and project objects
WITH admin AS (
    SELECT
        id
    FROM
        users
    WHERE
        username = 'admin'
    LIMIT 1
),
worker AS (
INSERT INTO workspaces (name, archived, user_id, IMMUTABLE)
    SELECT
        'Uncategorized',
        FALSE,
        admin.id,
        TRUE
    FROM
        admin
    RETURNING
        workspaces.id
),
newp AS (
INSERT INTO projects (name, description, archived, workspace_id, user_id, IMMUTABLE)
    SELECT
        'Uncategorized',
        '',
        FALSE,
        worker.id,
        admin.id,
        TRUE
    FROM
        admin,
        worker
    RETURNING
        projects.id)
UPDATE
    experiments
SET
    project_id = (
        SELECT
            id
        FROM
            newp);

