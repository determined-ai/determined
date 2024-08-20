/* Add RBAC permissions for creating/updating/deleting namespace-workspace bindings and 
resource quotas. */
INSERT into permissions(id, name, global_only) VALUES 
    (11003, 'view resource quotas', false);

INSERT INTO permission_assignments(permission_id, role_id) VALUES
    (11003, 1),
    (11003, 2),
    (11003, 4),
    (11003, 5),
    (11003, 7),
    (11003, 8),
    (11003, 9);
