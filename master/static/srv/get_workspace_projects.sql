SELECT
  p.id,
  p.name,
  p.workspace_id,
  p.description,
  p.immutable,
  p.notes,
  'WORKSPACE_STATE_' || p.state AS state,
  p.error_message,
  (w.archived OR p.archived) AS archived,
  SUM(CASE WHEN pe.project_id = p.id THEN 1 ELSE 0 END) AS num_experiments,
  SUM(
    CASE WHEN pe.project_id = p.id
    AND pe.state = 'ACTIVE' THEN 1 ELSE 0 END
  ) AS num_active_experiments,
  MAX(
    CASE WHEN pe.project_id = p.id THEN pe.start_time ELSE NULL END
  ) AS last_experiment_started_at,
  u.username,
  p.user_id
FROM
  projects AS p
  LEFT JOIN workspaces AS w ON p.workspace_id = w.id
  LEFT JOIN experiments AS pe ON p.id = pe.project_id
  LEFT JOIN users AS u ON u.id = p.user_id
WHERE
  ($1 = 0 OR p.workspace_id = $1)
  AND ($2 = '' OR (u.username IN (SELECT unnest(string_to_array($2, ',')))))
  AND ($3 = '' OR p.user_id IN (SELECT unnest(string_to_array($3, ',')::int [])))
  AND ($4 = '' OR p.name ILIKE $4)
  AND ($5 = '' OR p.archived = $5::BOOL)
GROUP BY
  p.id,
  u.username,
  p.user_id,
  w.archived
ORDER BY
  %s;
