INSERT INTO permissions(id, name, global_only) VALUES
    (4007, 'set default resource pool on workspace', false),
    (10001, 'modify rp workspace bindings', true);

-- add permission for cluster admin and workspace admin
INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 1 FROM permissions WHERE name IN (
    'set default resource pool on workspace',
    'modify rp workspace bindings'
);

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 2 FROM permissions WHERE name IN (
    'set default resource pool on workspace'
);

