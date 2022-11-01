INSERT INTO permissions(id, name, global_only) VALUES
    (4006, 'set checkpoint storage config on workspace', false);
INSERT INTO permission_assignments(permission_id, role_id) VALUES
    (4006, 1),
    (4006, 2);
