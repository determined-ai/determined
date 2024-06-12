/* Add RBAC permissions for creating/updating/deleting namespace-workspace bindings and 
resource quotas. */
INSERT into permissions(id, name, global_only) VALUES 
    (11001, 'set workspace-namespace bindings', true),
    (11002, 'set resource quotas', true);

INSERT INTO permission_assignments(permission_id, role_id)
SELECT permissions.id, 1 FROM permissions WHERE permissions.name IN (
    'set workspace-namespace bindings',
    'set resource quotas'
);
