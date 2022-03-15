SELECT p.id, p.name, p.workspace_id, p.description, p.archived, p.notes,
  COUNT(pe) AS num_experiments,
  SUM(case when pe.state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments,
  MAX(pe.start_time) AS last_experiment_started_at,
  u.username
FROM projects as p
  LEFT JOIN users as u ON u.id = p.user_id
  LEFT JOIN experiments pe ON pe.project_id = p.id
WHERE p.id = $1
GROUP BY p.id, pe.project_id, u.username;
