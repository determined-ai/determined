CREATE TYPE workload_type AS ENUM ('EXPERIMENT', 'NTSC');

CREATE TABLE IF NOT EXISTS task_config_policies
(
    workspace_id integer UNIQUE,
    last_updated_by integer NOT NULL,
    last_updated_time timestamptz NOT NULL DEFAULT now(),
    workload_type workload_type NOT NULL,
    invariant_config jsonb,
    constraints jsonb,

    CONSTRAINT fk_wksp_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id),
    CONSTRAINT pk_wksp_id_wksp_type PRIMARY KEY(workspace_id, workload_type),
    CONSTRAINT fk_last_updated_by_user_id FOREIGN KEY(last_updated_by) REFERENCES users(id)
);

