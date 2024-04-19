INSERT into permissions(id, name, global_only) VALUES 
    (7001, 'view model registry', false), 
    (7002, 'edit model registry', false), 
    (7003, 'create model registry', false); 

INSERT INTO roles(id, role_name) VALUES
    (6, 'ModelRegistryViewer'); 

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 1 FROM permissions WHERE name IN (
    'view model registry', 
    'edit model registry', 
    'create model registry');

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 2 FROM permissions WHERE name IN (
    'view model registry', 
    'edit model registry', 
    'create model registry');

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 4 FROM permissions WHERE name = 'view model registry'; 

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 5 FROM permissions WHERE name IN (
    'view model registry', 
    'edit model registry', 
    'create model registry');

INSERT INTO permission_assignments(permission_id, role_id)
SELECT id AS permission_id, 6 FROM permissions WHERE name = 'view model registry';
