INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 5 FROM permissions WHERE name IN (
    'update workspace'
);
