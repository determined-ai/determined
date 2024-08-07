INSERT INTO roles(id, role_name) VALUES
    (9, 'EditorProjectRestricted');

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 9 FROM permissions WHERE name IN (
    'create experiment',
    'update experiment',
    'view project',
    'update experiment metadata',
    'view experiment artifacts',
    'view experiment metadata',
    'delete experiment',
    'view workspace',
    'view model registry',
    'edit model registry',
    'create model registry',
    'create notebooks/shells/commands',
    'view notebooks/shells/commands',
    'update notebooks/shells/commands',
    'delete model registry',
    'view templates',
    'update templates',
    'create templates',
    'delete templates',
    'delete model version'
);
