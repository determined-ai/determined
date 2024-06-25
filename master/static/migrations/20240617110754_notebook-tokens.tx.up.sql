/*
   Create a new table for storing tokens for Jupyter notebook tasks.
*/

CREATE TABLE IF NOT EXISTS notebook_sessions (
    id SERIAL PRIMARY KEY,
    user_session_id INT NOT NULL REFERENCES user_sessions(id),
    task_id text NOT NULL UNIQUE REFERENCES tasks(task_id),
    token text NOT NULL
);
