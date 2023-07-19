WITH p AS (
  SELECT id FROM projects
  WHERE workspace_id = $1
),
exp_count AS (
  SELECT COUNT(*) AS count FROM experiments
  WHERE project_id IN (SELECT id FROM p)
)
SELECT w.id, w.name, w.archived, w.immutable, u.username, w.user_id, w.checkpoint_storage_config,
  'WORKSPACE_STATE_' || w.state AS state, w.error_message, w.default_compute_pool, w.default_aux_pool,
  (CASE WHEN uid IS NOT NULL OR gid IS NOT NULL OR user_ IS NOT NULL OR group_ IS NOT NULL THEN
    jsonb_build_object('agent_uid', uid, 'agent_user', user_, 'agent_gid', gid, 'agent_group', group_)
    ELSE NULL END) as agent_user_group,
  (SELECT COUNT(*) FROM p) AS num_projects,
  (SELECT count FROM exp_count) AS num_experiments,
  (SELECT COUNT(*) > 0 FROM workspace_pins
    WHERE workspace_id = $1 AND user_id = $2
  ) AS pinned
FROM workspaces as w
  LEFT JOIN users as u ON u.id = w.user_id
WHERE w.id = $1;
