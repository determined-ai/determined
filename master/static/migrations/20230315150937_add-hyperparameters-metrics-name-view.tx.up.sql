CREATE MATERIALIZED VIEW hyperparameters_view AS
    SELECT workspaces.id AS workspace_id,  experiments.project_id AS project_id, experiments.id AS experiment_id, json_build_array((experiments.config->'hyperparameters')) AS hyperparameters FROM workspaces, experiments, projects where workspaces.id = projects.workspace_id AND experiments.project_id = projects.id;

CREATE INDEX ix_hyperparameters_workspace_id ON hyperparameters_view USING btree (workspace_id)
CREATE INDEX ix_hyperparameters_project_id ON hyperparameters_view USING btree (project_id)


CREATE MATERIALIZED VIEW metrics_name_view AS  
    SELECT workspaces.id AS workspace_id,  experiments.project_id AS project_id, experiments.id AS experiment_id, 
