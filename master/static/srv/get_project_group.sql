SELECT g.id, g.name, g.project_id
FROM project_experiment_groups g
WHERE g.id = $1
