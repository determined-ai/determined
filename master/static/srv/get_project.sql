WITH pe AS (
  SELECT project_id, state, start_time
  FROM experiments
  WHERE project_id = $1
)
SELECT p.id, p.name, p.workspace_id, p.description, p.archived, p.notes,
  COUNT(pe) AS num_experiments,
  SUM(case when pe.state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments,
  MAX(pe.start_time) AS last_experiment_started_at,
  u.username
FROM pe, projects as p
  LEFT JOIN users as u ON u.id = p.user_id
WHERE p.id = $1
GROUP BY p.id, u.username;
