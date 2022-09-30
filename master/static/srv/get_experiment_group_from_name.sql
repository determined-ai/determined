SELECT g.id, g.name, g.project_id
FROM experiment_groups g
WHERE g.project_id = $1 AND g.name = $2;
