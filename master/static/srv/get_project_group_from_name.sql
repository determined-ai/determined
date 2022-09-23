SELECT g.id, g.name, g.project_id
FROM project_experiment_groups g
JOIN experiments e ON e.group_id = g.id
WHERE g.project_id = $1 AND g.name = $2;
