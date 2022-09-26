INSERT INTO permission_assignments(permission_id, role_id)
    SELECT id AS permission_id, 2 FROM permissions WHERE name = 'update project';
