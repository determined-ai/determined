/* Add RBAC permissions for creating/updating and viewing config policies. */
INSERT into permissions(id, name, global_only) VALUES 
    (11004, 'modify global config polcies', false),
    (11005, 'modify workspace config policies', false),
    (11006, 'view global config polcies', false),
    (11007, 'view workspace config polcies', false);

INSERT INTO permission_assignments(permission_id, role_id) VALUES
    (11004, 1),
    (11004, 2),
    (11005, 1),
    (11005, 2),
    (11006, 1),
    (11006, 2),
    (11006, 3),
    (11006, 4),
    (11006, 5),
    (11006, 6),
    (11006, 7),
    (11006, 8),
    (11006, 9),
    (11007, 1),
    (11007, 2),
    (11007, 3),
    (11007, 4),
    (11007, 5),
    (11007, 6),
    (11007, 7),
    (11007, 8),
    (11007, 9);
