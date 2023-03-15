CREATE MATERIALIZED VIEW hyperparameters_view AS
    SELECT workspace_id,  project_id, experiments.id AS experiment_id, hyperparameters FROM
