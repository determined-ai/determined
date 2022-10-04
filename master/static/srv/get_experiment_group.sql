WITH ge AS (
  SELECT COUNT(*) AS num_experiments
  FROM experiments
  WHERE group_id = $1
)
SELECT g.id, g.name, g.project_id, ge.num_experiments AS num_experiments
FROM ge, experiment_groups g
WHERE g.id = $1
