INSERT INTO permissions (
    id, name, global_only
) VALUES
(7004, 'delete model registry', false);
-- ClusterAdmin, WorkspaceAdmin, Editor roles
INSERT INTO permission_assignments (permission_id, role_id) VALUES
(7004, 1),
(7004, 2),
(7004, 5);