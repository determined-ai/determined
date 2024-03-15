INSERT INTO permissions (
    id, name, global_only
) VALUES
(8003, 'update agents', true);
-- ClusterAdmin role
INSERT INTO permission_assignments (permission_id, role_id) VALUES
(8003, 1);
