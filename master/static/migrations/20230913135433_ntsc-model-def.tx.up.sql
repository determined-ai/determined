CREATE TABLE task_context_directory (
    task_id text REFERENCES tasks(task_id) ON DELETE CASCADE NOT NULL UNIQUE PRIMARY KEY,
    context_directory bytea NOT NULL
);
