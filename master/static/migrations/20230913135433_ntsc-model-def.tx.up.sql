CREATE TABLE ntsc_model_definition (
    task_id text REFERENCES tasks(task_id) ON DELETE CASCADE NOT NULL UNIQUE PRIMARY KEY,
    model_definition bytea NOT NULL
);
