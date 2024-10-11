DELETE FROM permissions
WHERE id IN (12001, 12002, 12003, 12004, 12005, 12006);

DELETE FROM permission_assignments
WHERE permission_id IN (
    SELECT p.id FROM permissions p 
    WHERE p.name IN (
        'administrate access token',
        'update own access token',
        'create long lived token',
        'create other long lived token',
        'view other access token',
        'view own access token'
    )
)
AND role_id = 1;

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
