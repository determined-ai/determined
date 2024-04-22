INSERT INTO role_assignment_scopes(scope_workspace_id)
SELECT NULL WHERE NOT EXISTS (
    SELECT * FROM role_assignment_scopes WHERE scope_workspace_id IS NULL
);

INSERT INTO role_assignments(role_id, group_id, scope_id)
WITH
    g AS (
        SELECT id FROM groups WHERE user_id = 1
    ),
    s AS (
        SELECT id FROM role_assignment_scopes WHERE scope_workspace_id IS NULL
    )
SELECT 1, g.id AS group_id, s.id AS scope_id FROM g, s
WHERE (
    -- Only assign ClusterAdmin to 'admin' if it is a fresh cluster installation.
    SELECT MIN(created_at) >= NOW() - INTERVAL '1 hour' FROM gopg_migrations WHERE version = '20200401000000'
);
