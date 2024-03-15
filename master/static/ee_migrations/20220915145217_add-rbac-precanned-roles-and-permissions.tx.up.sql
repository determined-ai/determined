INSERT INTO permissions(id, name, global_only) VALUES
    (2001, 'create experiment', false),
    (2004, 'update experiment', false),
    (5002, 'view project', false),
    (96001, 'update roles', true),
    (2005, 'update experiment metadata', false),
    (93001, 'update group', true),
    (94001, 'create workspace', true),
    (4004, 'delete workspace', false),
    (5003, 'update project', false),
    (5004, 'delete project', false),
    (6002, 'assign roles', false),
    (91001, 'administrate user', true),
    (2002, 'view experiment artifacts', false),
    (4003, 'update workspace', false),
    (2003, 'view experiment metadata', false),
    (2006, 'delete experiment', false),
    (4002, 'view workspace', false),
    (5001, 'create project', false);

INSERT INTO roles(id, role_name) VALUES
    (1, 'ClusterAdmin'),
    (2, 'WorkspaceAdmin'),
    (3, 'WorkspaceCreator'),
    (4, 'Viewer'),
    (5, 'Editor');

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 1 FROM permissions;

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 2 FROM permissions WHERE name IN (
    'create experiment',
    'view experiment artifacts',
    'view experiment metadata',
    'update experiment',
    'update experiment metadata',
    'delete experiment',
    'view workspace',
    'update workspace',
    'delete workspace',
    'view project',
    'create project',
    'delete project',
    'assign roles'
);

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 3 FROM permissions WHERE name = 'create workspace';

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 4 FROM permissions WHERE name IN (
    'view experiment artifacts',
    'view experiment metadata',
    'view project',
    'view experiment'    
);

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 5 FROM permissions WHERE name IN (
    'create experiment',
    'view experiment artifacts',
    'view experiment metadata',
    'update experiment',
    'update experiment metadata',
    'delete experiment',
    'view workspace',
    'update workspace',
    'view project',
    'create project',
    'update project'
);
