SELECT *
FROM project_experiment_groups g
WHERE ($1 = 0) OR (g.project_id = $1)
