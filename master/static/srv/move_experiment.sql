WITH update_mn AS (
    UPDATE exp_metrics_name SET project_id = $2 WHERE experiment_id = $1
), update_exp AS (
    UPDATE experiments SET project_id = $2
    WHERE id = $1
    RETURNING id
), update_nhps AS (
    WITH recursive flat (key, value) AS (
		SELECT key, value
		FROM experiments,
		jsonb_each(config -> 'hyperparameters')
		WHERE id = $1
	UNION
		SELECT concat(f.key, '.', j.key), j.value
		FROM flat f,
		jsonb_each(f.value) j
		WHERE jsonb_typeof(f.value) = 'object' AND f.value -> 'type' IS NULL
	), flatten AS (
	SELECT key AS data
	FROM flat WHERE value -> 'type' IS NOT NULL 
	UNION SELECT jsonb_array_elements_text(hyperparameters) FROM projects WHERE id = $2
	), agg AS (
		SELECT array_to_json(array_agg(DISTINCT flatten.data)) AS adata FROM flatten
	) 
	UPDATE "projects" SET hyperparameters = agg.adata FROM agg WHERE (id = $2)
), update_hps AS (
    WITH recursive flat (key, value) AS (
        SELECT key, value
        FROM experiments,
        jsonb_each(config -> 'hyperparameters')
        WHERE project_id = $3 AND id <> $1
    UNION
        SELECT concat(f.key, '.', j.key), j.value
        FROM flat f,
        jsonb_each(f.value) j
        WHERE jsonb_typeof(f.value) = 'object' AND f.value -> 'type' IS NULL
    ), flatten AS (
    SELECT array_to_json(array_agg(DISTINCT key)) AS data
    FROM flat
    WHERE value -> 'type' IS NOT NULL)
    UPDATE projects SET hyperparameters = flatten.data FROM flatten WHERE id = $3
)
SELECT id FROM update_exp
