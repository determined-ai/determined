SELECT *
FROM experiment_groups g
WHERE ($1 = 0) OR (g.project_id = $1)
