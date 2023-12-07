--  // Ability to view external jobs
--  PERMISSION_TYPE_VIEW_EXTERNAL_JOBS = 8007;

INSERT into permissions(id, name, global_only) VALUES
    (8007, 'view external jobs', true);


-- determined> select * from roles;
-- +----+---------------------+----------------------------+
-- | id | role_name           | created_at                 |
-- |----+---------------------+----------------------------|
-- | 1  | ClusterAdmin        | 2023-05-30 16:20:54.825443 |
-- | 2  | WorkspaceAdmin      | 2023-05-30 16:20:54.825443 |
-- | 3  | WorkspaceCreator    | 2023-05-30 16:20:54.825443 |
-- | 4  | Viewer              | 2023-05-30 16:20:54.825443 |
-- | 5  | Editor              | 2023-05-30 16:20:54.825443 |
-- | 6  | ModelRegistryViewer | 2023-05-30 16:20:55.136146 |
-- +----+---------------------+----------------------------+
-- SELECT 6

INSERT INTO permission_assignments (permission_id, role_id)
SELECT 8007, roles.id
FROM roles
WHERE roles.role_name IN ('ClusterAdmin');