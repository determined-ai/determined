INSERT INTO permissions (id, name, global_only) VALUES
    (12001, 'revoke long lived token', true),
    (12002, 'revoke other long lived token', true),
    (12003, 'create long lived token', true),
    (12004, 'create other long lived token', true),
    (12005, 'view long lived token', true),
    (12006, 'view other long lived token', true)
    ON CONFLICT DO NOTHING;

INSERT INTO permission_assignments (permission_id, role_id) VALUES
    (12001, 1), --	ClusterAdmin -- revoke long lived token
    (12001, 2), --	WorkspaceAdmin
    (12001, 5), --	Editor
    (12001, 7), --	EditorRestricted
    (12001, 8), --	GenAIUser
    (12001, 9), --	EditorProjectRestricted
    (12002, 1), --	ClusterAdmin -- revoke other long lived token
    (12003, 1), --	ClusterAdmin -- create long lived token
    (12003, 2), --	WorkspaceAdmin
    (12003, 5), --	Editor
    (12003, 7), --	EditorRestricted
    (12003, 8), --	GenAIUser
    (12003, 9), --	EditorProjectRestricted
    (12004, 1), --	ClusterAdmin -- create other long lived token
    (12005, 1), --	ClusterAdmin -- view long lived token
    (12005, 2), --	WorkspaceAdmin
    (12005, 3), -- WorkspaceCreator
    (12005, 4), -- Viewer
    (12005, 5), --	Editor
    (12005, 6), -- ModelRegistryViewer
    (12005, 7), --	EditorRestricted
    (12005, 8), --	GenAIUser
    (12005, 9), --	EditorProjectRestricted
    (12006, 1) --	ClusterAdmin -- view other long lived token
    ON CONFLICT DO NOTHING;


