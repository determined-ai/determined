INSERT INTO permissions (
    id, name, global_only
) VALUES
(8004, 'view sensitive agent info', true);
-- ClusterAdmin role
INSERT INTO permission_assignments (permission_id, role_id) VALUES (8004, 1);
