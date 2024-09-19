/* Remove modify global config policies RBAC permission from WorkspaceAdmin. */
DELETE FROM permission_assignments WHERE permission_id = 11004 and role_id = 2;

/* Assign global_only to global RBAC permission. */
UPDATE permissions SET global_only = true WHERE id = 11004;
