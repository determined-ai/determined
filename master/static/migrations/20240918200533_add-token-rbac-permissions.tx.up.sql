INSERT INTO permissions (id, name, global_only) VALUES
    (12001, 'administrate access token', true),
    (12002, 'update own access token', true),
    (12003, 'create long lived token', true),
    (12004, 'create other long lived token', true),
    (12005, 'view other access token', true),
    (12006, 'view own access token', true);
    

-- ClusterAdmin
INSERT INTO permission_assignments (permission_id, role_id) 
SELECT p.id AS permission_id, 1 FROM permissions p WHERE p.name IN (
    'administrate access token',
    'update own access token',
    'create long lived token',
    'create other long lived token',
    'view other access token',
    'view own access token'
);

-- WorkspaceAdmin, Editor, EditorRestricted, GenAIUser, EditorProjectRestricted
INSERT INTO permission_assignments (permission_id, role_id)
SELECT p.id AS permission_id, r.id FROM permissions p JOIN roles r ON r.role_name IN (
    'WorkspaceAdmin',
    'Editor',
    'EditorRestricted',
    'GenAIUser',
    'EditorProjectRestricted'
) WHERE p.name IN (
    'update own access token',
    'create long lived token',
    'view own access token'
);

-- Viewer, ModelRegistryViewer, WorkspaceCreator
INSERT INTO permission_assignments (permission_id, role_id)
SELECT p.id AS permission_id, r.id FROM permissions p JOIN roles r ON r.role_name IN (
    'Viewer',
    'ModelRegistryViewer',
    'WorkspaceCreator'
) WHERE p.name IN (
    'view own access token'
);
