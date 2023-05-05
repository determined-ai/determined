CREATE TABLE workspace_pins (
    id serial PRIMARY KEY,
    workspace_id int REFERENCES workspaces (id) ON DELETE CASCADE,
    user_id int REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, workspace_id)
);

CREATE INDEX ix_workspace_pins ON public.workspace_pins USING btree (workspace_id);

