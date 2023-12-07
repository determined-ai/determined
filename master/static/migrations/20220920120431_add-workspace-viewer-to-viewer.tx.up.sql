INSERT INTO permission_assignments(permission_id, role_id)
    SELECT id AS permission_id, 4 FROM permissions WHERE name = 'view workspace';
