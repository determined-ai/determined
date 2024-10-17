INSERT INTO roles(id, role_name) VALUES
    (10, 'TokenCreator');

INSERT INTO permissions (id, name, global_only) VALUES
    (12001, 'administrate access token', true),
    (12002, 'update own access token', true),
    (12003, 'create access token', true),
    (12004, 'create other access token', true),
    (12005, 'view other access token', true),
    (12006, 'view own access token', true);
    

-- ClusterAdmin
INSERT INTO permission_assignments (permission_id, role_id) 
SELECT p.id AS permission_id, 1 FROM permissions p WHERE p.name IN (
    'administrate access token',
    'update own access token',
    'create access token',
    'create other access token',
    'view other access token',
    'view own access token'
);

-- TokenCreator
INSERT INTO permission_assignments (permission_id, role_id) 
SELECT p.id AS permission_id, 10 FROM permissions p WHERE p.name IN (
    'update own access token',
    'create access token',
    'view own access token'
);
