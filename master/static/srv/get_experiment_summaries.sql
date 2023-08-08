SELECT
    id,
    job_id,
    state,
    config,
    owner_id,
    progress,
    archived,
    start_time,
    end_time
FROM
    experiments
WHERE
    state IN (SELECT unnest(string_to_array($1, ','))::experiment_state)
ORDER BY
    id DESC
