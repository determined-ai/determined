DELETE FROM role_assignments WHERE group_id IN (
    SELECT id FROM groups WHERE user_id = 1
) AND role_id = 1;
