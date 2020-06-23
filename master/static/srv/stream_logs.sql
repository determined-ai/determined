SELECT
    l.id as id,
    encode(l.message, 'escape') as message
FROM trial_logs l
WHERE l.trial_id = $1
ORDER BY l.id ASC OFFSET $2 LIMIT $3
