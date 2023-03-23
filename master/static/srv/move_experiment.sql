WITH update_mn AS (
    UPDATE exp_metrics_name SET project_id = $2 WHERE experiment_id = $1
), update_exp AS (
    UPDATE experiments SET project_id = $2
    WHERE id = $1
    RETURNING id
)
SELECT id FROM update_exp


