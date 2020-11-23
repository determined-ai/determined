SELECT
    t.state AS State,
    (SELECT count(*)
        FROM trial_logs l
        WHERE l.trial_id = $1) AS NumLogs
FROM trials t
WHERE t.id = $1
