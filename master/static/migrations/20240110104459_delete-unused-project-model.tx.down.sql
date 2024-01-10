CREATE TABLE project_models (
  name TEXT,
  checkpoint_id INT,
  created_at timestamp with time zone NOT NULL DEFAULT NOW(),
  project_id INT REFERENCES projects(id)
);

CREATE INDEX ix_project_models_project_id ON project_models USING btree (project_id);
