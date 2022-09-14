INSERT INTO groups(group_name, user_id)
SELECT username || 'DeterminedPersonalGroup' as group_name, id AS user_id FROM users;

INSERT INTO user_group_membership(user_id, group_id)
SELECT user_id AS user_id, id as group_id FROM groups
    WHERE group_name LIKE '%DeterminedPersonalGroup';
