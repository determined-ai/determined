CREATE MATERIALIZED VIEW hyperparameters_view AS
    SELECT workspaces.id AS workspace_id,  experiments.project_id AS project_id, experiments.id AS experiment_id, json_build_array((experiments.config->'hyperparameters')) AS hyperparameters, FROM workspaces, experiments, projects where workspaces.id = projects.workspace_id AND experiments.project_id = projects.id;

CREATE INDEX ix_hyperparameters_workspace_id ON hyperparameters_view USING btree (workspace_id)
CREATE INDEX ix_hyperparameters_project_id ON hyperparameters_view USING btree (project_id)


CREATE MATERIALIZED VIEW metrics_name_view AS  
    SELECT workspaces.id AS workspace_id,  experiments.project_id AS project_id, experiments.id AS experiment_id, array_agg(DISTINCT names) AS name FROM workspaces, experiments, projects, steps, trials, 
    LATERAL jsonb_object_keys(steps.metrics->'avg_metrics') AS names
where workspaces.id = projects.workspace_id AND experiments.project_id = projects.id AND trials.id=steps.trial_id 
group by (workspaces.id, experiments.project_id, experiments.id);


with training_metrics_names AS (SELECT
   array_agg(DISTINCT names) AS name, e.id as experiment_id
FROM trials t, experiments e, steps s,
    LATERAL jsonb_object_keys(s.metrics->'avg_metrics') AS names
where t.id=s.trial_id and e.id = t.experiment_id 
group by e.id)
SELECT workspaces.id AS workspace_id,  experiments.project_id AS project_id, experiments.id AS experiment_id, training_metrics_names.name FROM workspaces, experiments, projects, training_metrics_names where workspaces.id = projects.workspace_id AND experiments.project_id = projects.id AND training_metrics_names.experiment_id = experiments.id and workspaces.id > 1; 

