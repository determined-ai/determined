CREATE TYPE workload_type AS ENUM ('EXPERIMENT', 'NTSC');

CREATE TABLE IF NOT EXISTS task_config_policies
(
    workspace_id integer,  
    workload_type workload_type NOT NULL,
    last_updated_by integer NOT NULL,
    last_updated_time timestamptz NOT NULL DEFAULT current_timestamp,
    invariant_config jsonb,
    constraints jsonb,

    CONSTRAINT fk_wksp_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_last_updated_by_user_id FOREIGN KEY(last_updated_by) REFERENCES users(id)
        ON DELETE CASCADE
);


CREATE UNIQUE INDEX task_config_policies_workload_type_idx ON task_config_policies (workload_type) 
    WHERE workspace_id IS NULL; 
CREATE UNIQUE INDEX wksp_id_wkld_type ON task_config_policies (workspace_id, workload_type) 
WHERE workspace_id IS NOT NULL;
