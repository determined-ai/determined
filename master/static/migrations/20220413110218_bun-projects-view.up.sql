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
