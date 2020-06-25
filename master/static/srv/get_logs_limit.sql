WITH logs AS (
    SELECT
        trial_logs.id as id,
        trials.state as state,
        encode(message, 'escape') as message
    FROM trial_logs JOIN trials
    ON trial_logs.trial_id = trials.id
    WHERE trials.id = $1
    ORDER BY trial_logs.id DESC
    LIMIT $2
)
SELECT * FROM logs ORDER BY id ASC;
