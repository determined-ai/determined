INSERT INTO permissions (
    id, name, global_only
) VALUES
(7005, 'delete model version', false),
(7006, 'delete other user model registry', false),
(7007, 'delete other user model version', false);
INSERT INTO permission_assignments (permission_id, role_id) VALUES
(7005, 1), -- ClusterAdmin
(7005, 2), -- WorkspaceAdmin
(7005, 5), -- Editor
(7006, 1),
(7006, 2),
(7007, 1),
(7007, 2);