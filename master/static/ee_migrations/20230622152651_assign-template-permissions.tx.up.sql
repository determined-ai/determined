--   // Ability to view templates.
--   PERMISSION_TYPE_VIEW_TEMPLATES = 9001;
--   // Ability to update templates.
--   PERMISSION_TYPE_UPDATE_TEMPLATES = 9002;
--   // Ability to create templates.
--   PERMISSION_TYPE_CREATE_TEMPLATES = 9003;
--   // Ability to delete templates.
--   PERMISSION_TYPE_DELETE_TEMPLATES = 9004;

INSERT INTO permissions(id, name, global_only) VALUES
    (9001, 'view templates', false),
    (9002, 'update templates', false),
    (9003, 'create templates', false),
    (9004, 'delete templates', false);

--    (1, 'ClusterAdmin'),
--    (2, 'WorkspaceAdmin'),
--    (3, 'WorkspaceCreator'),
--    (4, 'Viewer'),
--    (5, 'Editor');

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 1 FROM permissions WHERE name IN (
    'view templates',
    'update templates',
    'create templates',
    'delete templates'
);

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 2 FROM permissions WHERE name IN (
    'view templates',
    'update templates',
    'create templates',
    'delete templates'
);

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 4 FROM permissions WHERE name IN (
    'view templates'
);

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 5 FROM permissions WHERE name IN (
    'view templates',
    'update templates',
    'create templates',
    'delete templates'
);
