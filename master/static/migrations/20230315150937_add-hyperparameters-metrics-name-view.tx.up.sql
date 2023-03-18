CREATE TABLE public.exp_hyperparameters (
    id SERIAL PRIMARY KEY,
    project_id INT REFERENCES projects(id),
    experiment_id INT REFERENCES experiments(id),
    hyperparameters JSON 
);

CREATE INDEX ix_hyperparameters_project_id ON exp_hyperparameters USING btree (project_id);
CREATE UNIQUE INDEX ix_hyperparameters_experiment_id_unique ON exp_hyperparameters(experiment_id);

INSERT INTO public.exp_hyperparameters (project_id, experiment_id, hyperparameters) (
    SELECT project_id, id AS experiment_id, json_build_array((config->'hyperparameters')) AS hyperparameters FROM experiments
);

CREATE OR REPLACE FUNCTION autoupdate_exp_hyperparameters() RETURNS trigger AS $$
BEGIN
  INSERT INTO exp_hyperparameters (project_id, experiment_id, hyperparameters) VALUES (
    NEW.project_id, NEW.id, json_build_array((NEW.config->'hyperparameters'))
  ) ON CONFLICT(experiment_id) DO UPDATE SET hyperparameters = EXCLUDED.hyperparameters;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_exp_hyperparameters
AFTER INSERT OR UPDATE ON experiments
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_hyperparameters();



CREATE TABLE public.exp_metrics_name (
    id SERIAL PRIMARY KEY,
    project_id INT REFERENCES projects(id),
    experiment_id INT REFERENCES experiments(id),
    tname JSON,
    vname JSON
);

CREATE INDEX ix_metrics_name_project_id ON exp_metrics_name USING btree (project_id);
CREATE UNIQUE INDEX ix_metrics_name_experiment_id_unique ON exp_metrics_name(experiment_id);

INSERT INTO public.exp_metrics_name (project_id, experiment_id, tname, vname) (
    WITH training_metrics_names AS (
        SELECT array_to_json(array_agg(DISTINCT names)) AS name, e.id AS experiment_id
        FROM trials t, experiments e, steps s,
            LATERAL jsonb_object_keys(s.metrics->'avg_metrics') AS names
        WHERE t.id=s.trial_id AND e.id = t.experiment_id 
        GROUP BY e.id),
    validation_metrics_names AS (
        SELECT array_to_json(array_agg(DISTINCT names)) AS name, e.id AS experiment_id
        FROM trials t, experiments e, validations v,
            LATERAL jsonb_object_keys(v.metrics->'validation_metrics') AS names
        WHERE t.id=v.trial_id AND e.id = t.experiment_id 
        GROUP BY e.id)
    SELECT   
        e.project_id AS project_id, 
        e.id AS experiment_id, 
        COALESCE(training_metrics_names.name, '[]'::json) AS tname,
        COALESCE(validation_metrics_names.name, '[]'::json) AS vname 
    FROM experiments e LEFT JOIN validation_metrics_names ON e.id = validation_metrics_names.experiment_id, training_metrics_names 
    WHERE 
        training_metrics_names.experiment_id = e.id
);

CREATE OR REPLACE FUNCTION autoupdate_exp_training_metrics_name() RETURNS trigger AS $$
BEGIN
    INSERT INTO exp_metrics_name (project_id, experiment_id, tname) (
        SELECT e.project_id, e.id, array_to_json(array_agg(DISTINCT names)) AS tname
        FROM experiments e, trials t, raw_steps s, LATERAL jsonb_object_keys(s.metrics->'avg_metrics') AS names
        WHERE  t.experiment_id = (SELECT experiment_id FROM trials WHERE id = NEW.trial_id) AND t.id = s.trial_id AND e.id=t.experiment_id GROUP BY (e.project_id, e.id)
    )  ON CONFLICT(experiment_id) DO UPDATE SET tname = EXCLUDED.tname;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_exp_training_metrics_name
AFTER INSERT ON raw_steps
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_training_metrics_name();

CREATE OR REPLACE FUNCTION autoupdate_exp_validation_metrics_name() RETURNS trigger AS $$
BEGIN
    INSERT INTO exp_metrics_name (project_id, experiment_id, vname) (
        SELECT e.project_id, e.id, array_to_json(array_agg(DISTINCT names)) AS vname
        FROM experiments e, trials t, raw_validations v, LATERAL jsonb_object_keys(v.metrics->'validation_metrics') AS names
        WHERE  t.experiment_id = (SELECT experiment_id FROM trials WHERE id = NEW.trial_id) AND t.id = v.trial_id AND e.id=t.experiment_id GROUP BY (e.project_id, e.id)
    )  ON CONFLICT(experiment_id) DO UPDATE SET vname = EXCLUDED.vname;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_exp_validation_metrics_name
AFTER INSERT ON raw_validations
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_validation_metrics_name();

