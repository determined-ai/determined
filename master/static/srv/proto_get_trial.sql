SELECT t.id,
    t.experiment_id,
    'STATE_' || t.state AS state,
    t.start_time,
    t.end_time,
    t.hparams,
    (
        SELECT s.prior_batches_processed + s.num_batches
        FROM steps s
        WHERE s.trial_id = t.id
            AND s.state = 'COMPLETED'
        ORDER BY s.id DESC
        LIMIT 1
    ) AS total_batches_processed
FROM trials t
WHERE t.id = $1
