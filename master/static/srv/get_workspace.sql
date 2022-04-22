SELECT w.id, w.name, w.archived, w.immutable, u.username,
  (SELECT COUNT(*) FROM projects WHERE workspace_id = $1) AS num_projects,
  (SELECT COUNT(*) > 0 FROM workspace_pins
    WHERE workspace_id = $1 AND user_id = $2
  ) AS pinned
FROM workspaces as w
  LEFT JOIN users as u ON u.id = w.user_id
WHERE w.id = $1;
