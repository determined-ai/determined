UPDATE permissions SET global_only = false WHERE ID = 97001;

INSERT INTO permissions(id, name, global_only) VALUES (97002, 'view webhooks', false) ON CONFLICT DO NOTHING;

INSERT INTO permission_assignments(permission_id, role_id) VALUES 
    (97001, 2), -- Workspace admin can edit webhooks
    (97001, 5), -- Editor can edit webhooks
    (97001, 7), -- EditorRestricted can edit webhooks
    (97001, 9), -- EditorProjectRestricted can edit webhooks
    (97002, 1), -- Cluster admin can view webhooks
    (97002, 2), -- Workspace admin can view webhooks
    (97002, 5), -- Editor can view webhooks
    (97002, 7), -- EditorRestricted can view webhooks
    (97002, 9), -- EditorProjectRestricted can view webhooks
    (97002, 4)  -- Viewer can view webhooks
    ON CONFLICT DO NOTHING; 
