SELECT t.id AS id,
    t.start_time AS start_time,
    t.end_time AS end_time,
    t.experiment_id AS experiment_id,
    t.hparams AS hparams,
    'STATE_' || t.state AS state,
    (
        SELECT s.prior_batches_processed + s.num_batches
        FROM steps s
        WHERE s.trial_id = t.id
            AND s.state = 'COMPLETED'
        ORDER BY s.id DESC
        LIMIT 1
    ) AS batches_processed
FROM trials t
WHERE t.experiment_id = $1
