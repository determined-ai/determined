INSERT INTO permissions (
    id, name, global_only
) VALUES
(8001, 'view master logs', true),
(8002, 'view detailed cluster usage', true);
-- ClusterAdmin role
INSERT INTO permission_assignments (permission_id, role_id) VALUES (8001, 1),
(8002, 1);
