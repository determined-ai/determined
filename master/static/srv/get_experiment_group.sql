SELECT g.id, g.name, g.project_id
FROM experiment_groups g
WHERE g.id = $1
