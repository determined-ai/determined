SELECT p.id, p.name, p.workspace_id, p.description, p.archived,
  COUNT(pe) AS num_experiments,
  SUM(case when pe.state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments,
  MAX(pe.start_time) AS last_experiment_started_at,
  my_notes AS notes,
  u.username
FROM projects as p
  LEFT JOIN users as u ON u.id = p.user_id
  LEFT JOIN notes my_notes ON my_notes.project_id = p.id
  LEFT JOIN experiments pe ON pe.project_id = p.id
  WHERE p.workspace_id = $1
  AND ($2 = '' OR (u.username IN (SELECT unnest(string_to_array($2, ',')))))
  AND ($3 = '' OR p.name ILIKE $3)
  AND ($4 = '' OR p.archived = $4::BOOL)
GROUP BY p.id, pe.project_id, my_notes.id, u.username
ORDER BY %s;
