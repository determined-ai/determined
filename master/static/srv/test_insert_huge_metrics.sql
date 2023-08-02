WITH ss AS (
    SELECT
        trial_id,
        end_time,
        metrics,
        total_batches,
        trial_run_id,
        archived
    FROM steps
    WHERE trial_id=$1
    ORDER BY total_batches DESC
    LIMIT 1
)

INSERT INTO steps
(trial_id, end_time, metrics, total_batches, trial_run_id, archived)
SELECT
    trial_id,
    end_time,
    metrics,
    total_batches+g,
    trial_run_id,
    archived
FROM ss, generate_series(1, $2) g
RETURNING false;
