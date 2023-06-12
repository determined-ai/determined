CREATE TABLE rp_workspace_bindings ( 
    workspace_id INT NOT NULL, 
    pool_name TEXT NOT NULL, 
    validity BOOLEAN NOT NULL,   
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE 
);

ALTER TABLE workspaces 
    ADD COLUMN default_pool TEXT DEFAULT NULL;  