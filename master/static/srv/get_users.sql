SELECT
	u.id, u.display_name, u.username, u.admin, u.active, u.modified_at,
	h.uid AS agent_uid, h.gid AS agent_gid, h.user_ AS agent_user, h.group_ AS agent_group, 
	COALESCE(u.display_name, u.username) AS name
FROM users u
LEFT OUTER JOIN agent_user_groups h ON (u.id = h.user_id)
ORDER BY %s;
