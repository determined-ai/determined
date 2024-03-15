INSERT INTO permissions(id, name, global_only) VALUES
    (3001, 'create notebooks/shells/commands', false),
    (3002, 'view notebooks/shells/commands', false),
    (3003, 'update notebooks/shells/commands', false);
INSERT INTO permission_assignments(permission_id, role_id) VALUES
    (3001, 1),
    (3001, 2),
    (3001, 5),
    (3002, 1),
    (3002, 2),
    (3002, 4),
    (3002, 5),
    (3003, 1),
    (3003, 2),
    (3003, 5);
