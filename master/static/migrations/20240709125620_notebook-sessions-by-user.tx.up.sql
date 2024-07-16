/*
   Changes the `notebook_sessions` table to associate with user IDs instead of user session IDs.
*/

-- Add a new user_id column.
ALTER TABLE notebook_sessions ADD COLUMN user_id int REFERENCES users(id) NULL;

-- Migrate existing rows.
UPDATE notebook_sessions ns
SET user_id = us.user_id
FROM user_sessions us
WHERE ns.user_session_id = us.id;

-- Add not null constraint.
ALTER TABLE notebook_sessions ALTER COLUMN user_id SET NOT NULL;

-- Drop previous column.
ALTER TABLE notebook_sessions DROP COLUMN user_session_id;

-- Drop unused token column.
ALTER TABLE notebook_sessions DROP COLUMN token;
