SELECT id, name, project_id, (SELECT COUNT(*) FROM experiments WHERE group_id = g.id) AS num_experiments
FROM experiment_groups g
WHERE g.project_id = $1
