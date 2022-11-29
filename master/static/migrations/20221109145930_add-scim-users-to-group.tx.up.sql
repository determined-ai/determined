INSERT INTO groups(group_name, user_id)
SELECT username || 'DeterminedPersonalGroup' AS group_name, id AS user_id FROM users
ON CONFLICT DO NOTHING;

INSERT INTO user_group_membership(user_id, group_id)
SELECT user_id AS user_id, id AS group_id FROM groups
WHERE group_name LIKE '%DeterminedPersonalGroup'
ON CONFLICT DO NOTHING;
