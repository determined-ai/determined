WITH active_experiments AS (
  SELECT * FROM experiments
  WHERE state = 'ACTIVE' AND project_id = $1
)
SELECT p.id, p.name, p.workspace_id, p.description, p.archived,
  COUNT(pe) AS num_experiments,
  COUNT(ape) AS num_active_experiments,
  MAX(pe.start_time) AS last_experiment_started_at,
  my_notes AS notes,
  u.username
FROM projects as p
  LEFT JOIN users as u ON u.id = p.user_id
  LEFT JOIN notes my_notes ON my_notes.project_id = p.id
  LEFT JOIN experiments pe ON pe.project_id = p.id
  LEFT JOIN active_experiments ape ON ape.project_id = p.id
WHERE p.id = $1
GROUP BY p.id, pe.project_id, ape.project_id, my_notes.id, u.username;
