DELETE FROM permission_assignments WHERE role_id = 4 AND permission_id =
    (SELECT id AS permission_id FROM permissions WHERE name = 'view workspace');
