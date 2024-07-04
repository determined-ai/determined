CREATE TABLE public.workspace_namespace_bindings (
    workspace_id INT REFERENCES workspaces(id) ON DELETE CASCADE,
    cluster_name text NOT NULL,
    namespace text NOT NULL,
    auto_create_namespace boolean,
    PRIMARY KEY(workspace_id, cluster_name, namespace)
);
