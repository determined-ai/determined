CREATE TYPE public.webhook_mode AS ENUM (
    'WORKSPACE',
    'SPECIFIC'
);

ALTER table public.webhooks 
    ADD COLUMN workspace_id INT REFERENCES workspaces(id) DEFAULT NULL,
    ADD COLUMN name text NOT NULL DEFAULT md5(random()::text),
    ADD COLUMN mode public.webhook_mode NOT NULL DEFAULT 'WORKSPACE';

CREATE UNIQUE INDEX name_workspace_key on public.webhooks (name, workspace_id);
