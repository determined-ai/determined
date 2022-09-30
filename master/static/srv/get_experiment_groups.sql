SELECT id, name, project_id
FROM experiment_groups g
WHERE g.project_id = $1
