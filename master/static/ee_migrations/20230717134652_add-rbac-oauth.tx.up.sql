--   // Ability to manage OAuth clients and settings.
--   PERMISSION_TYPE_ADMINSTRATE_OAUTH = 91002;
INSERT INTO permissions(id, name, global_only) VALUES
(91002, 'manage oauth', true);

INSERT INTO permission_assignments(permission_id, role_id)
 SELECT p.id AS permission_id, r.id AS role_id
 FROM permissions p
 JOIN roles r ON r.role_name = 'ClusterAdmin'
 WHERE p.name IN ('manage oauth');