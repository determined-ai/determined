SELECT
    t.state AS State,
    COUNT(*) AS NumLogs
FROM trials t
JOIN trial_logs l ON t.id = l.trial_id
WHERE t.id = $1
GROUP BY t.state
