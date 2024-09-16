INSERT INTO permissions (
    id, name, global_only
) VALUES
(12001, 'revoke long lived token', true),
INSERT INTO permission_assignments (permission_id, role_id) VALUES
(12001, 1), --	ClusterAdmin
(12001, 2), --	WorkspaceAdmin
(12001, 3), --	WorkspaceCreator
(12001, 5), --	Editor
(12001, 7), --	EditorRestricted
(12001, 8), --	GenAIUser
(12001, 9); --	EditorProjectRestricted
