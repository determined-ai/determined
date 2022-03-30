SELECT w.id, w.name, w.archived, w.immutable, u.username
FROM workspaces as w
LEFT JOIN users as u ON u.id = w.user_id
WHERE ($1 = '' OR (u.username IN (SELECT unnest(string_to_array($1, ',')))))
AND ($2 = '' OR w.name ILIKE $2)
AND ($3 = '' OR w.archived = $3::BOOL)
ORDER BY %s;
