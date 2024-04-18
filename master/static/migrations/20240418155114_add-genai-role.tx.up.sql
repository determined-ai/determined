INSERT INTO roles(id, role_name) VALUES
    (8, 'GenAIUser');

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 8 FROM permissions WHERE name IN (
    'create experiment',
    'view experiment artifacts',
    'view experiment metadata',
    'update experiment',
    'update experiment metadata',
    'delete experiment',
    'create notebooks/shells/commands',
    'view notebooks/shells/commands',
    'update notebooks/shells/commands',
    'create workspace',
    'view workspace',
    'create project',
    'view project',
    'update project',
    'delete project',
    'view model registry',
    'edit model registry',
    'create model registry',
    'view templates',
    'update templates',
    'create templates',
    'delete templates'
);
