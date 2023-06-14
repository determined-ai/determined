DROP TABLE rp_workspace_bindings;

ALTER table workspaces
    DROP COLUMN default_compute_pool,
    DROP COLUMN default_aux_pool;
