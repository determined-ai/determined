DROP VIEW IF EXISTS public.projects_view;
CREATE VIEW public.projects_view AS
  SELECT p.id, p.name, p.workspace_id, p.description, p.immutable, p.notes,
    (p.archived OR w.archived) AS archived,
    COUNT(pe) AS num_experiments,
    SUM(case when pe.state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments,
    COALESCE(MAX(pe.start_time), NULL) AS last_experiment_started_at,
    u.username
  FROM projects as p
    LEFT JOIN users as u ON u.id = p.user_id
    LEFT JOIN workspaces AS w on w.id = p.workspace_id
    LEFT JOIN experiments AS pe ON pe.project_id = p.id
  GROUP BY p.id, u.username, w.archived;

DROP VIEW IF EXISTS public.experiments_view;
CREATE VIEW public.experiments_view AS
  SELECT
      e.id AS id,
      e.config->>'name' AS name,
      e.config->>'description' AS description,
      e.config->'labels' AS labels,
      e.config->'resources'->>'resource_pool' as resource_pool,
      e.config->'searcher'->'name' as searcher_type,
      e.notes AS notes,
      e.start_time AS start_time,
      e.end_time AS end_time,
      'STATE_' || e.state AS state,
      e.archived AS archived,
      e.progress AS progress,
      e.job_id AS job_id,
      e.parent_id AS forked_from,
      e.owner_id AS user_id,
      u.username AS username,
      e.project_id AS project_id
  FROM
      experiments e
  JOIN users u ON e.owner_id = u.id
