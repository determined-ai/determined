INSERT INTO permissions(id, name, global_only) VALUES
    (93002, 'create group', true),
    (93003, 'delete group', true);

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 1 FROM permissions WHERE name IN (
    'create group',
    'delete group',
);