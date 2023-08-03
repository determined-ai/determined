CREATE OR REPLACE FUNCTION autoupdate_exp_best_trial_metrics() RETURNS trigger AS $$
BEGIN
    WITH bt AS (SELECT id, best_validation_id FROM trials WHERE experiment_id = NEW.experiment_id ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE experiments SET best_trial_id = bt.id, 
    validation_metrics = 
    (SELECT metrics -> 'validation_metrics' FROM validations v WHERE v.id = bt.best_validation_id) FROM bt
    WHERE experiments.id = NEW.experiment_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER autoupdate_exp_best_trial_metrics
AFTER UPDATE OF best_validation_id ON trials
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_best_trial_metrics();

CREATE TABLE public.exp_metrics_name (
    id SERIAL PRIMARY KEY,
    project_id INT REFERENCES projects(id) ON DELETE CASCADE NOT NULL,
    experiment_id INT REFERENCES experiments(id) ON DELETE CASCADE NOT NULL,
    vname JSON
);

CREATE INDEX ix_metrics_name_project_id ON exp_metrics_name USING btree (project_id);
CREATE UNIQUE INDEX ix_metrics_name_experiment_id_unique ON exp_metrics_name(experiment_id);

INSERT INTO public.exp_metrics_name (project_id, experiment_id, vname) (
    WITH validation_metrics_names AS (
        SELECT array_to_json(array_agg(DISTINCT names)) AS name, e.id AS experiment_id
        FROM trials t, experiments e, raw_validations v,
            LATERAL jsonb_object_keys(v.metrics->'validation_metrics') AS names
        WHERE t.best_validation_id=v.id AND e.id = t.experiment_id 
        GROUP BY e.id)
    SELECT   
        e.project_id AS project_id, 
        e.id AS experiment_id, 
        COALESCE(validation_metrics_names.name, '[]'::json) AS vname 
    FROM experiments e, validation_metrics_names
    WHERE 
        validation_metrics_names.experiment_id = e.id
);

CREATE OR REPLACE FUNCTION autoupdate_exp_validation_metrics_name() RETURNS trigger AS $$
BEGIN
    WITH mname AS (
        SELECT array_to_json(array_agg(DISTINCT names)) AS mdata
        FROM LATERAL (SELECT jsonb_object_keys(NEW.metrics->'validation_metrics') AS names
        UNION SELECT json_array_elements_text(vname) FROM exp_metrics_name WHERE experiment_id = (SELECT experiment_id FROM trials WHERE id = NEW.trial_id)) AS foo
    )
    INSERT INTO exp_metrics_name (project_id, experiment_id, vname) (
        SELECT e.project_id, e.id, mname.mdata AS vname
        FROM experiments e, trials t, mname
        WHERE t.id = NEW.trial_id AND e.id = t.experiment_id 
    )  ON CONFLICT(experiment_id) DO UPDATE SET vname = EXCLUDED.vname;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_exp_validation_metrics_name
AFTER INSERT ON raw_validations
FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_validation_metrics_name();