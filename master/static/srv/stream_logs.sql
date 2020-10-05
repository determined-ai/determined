SELECT
    l.id,
    l.trial_id,
    encode(l.message, 'escape') as message,
    l.agent_id,
    l.container_id,
    l.rank_id,
    l.timestamp,
    l.level,
    l.std_type,
    l.source
FROM trial_logs l
WHERE l.trial_id = $1
ORDER BY l.id ASC OFFSET $2 LIMIT $3
