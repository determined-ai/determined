CREATE TYPE token_type AS ENUM ('USER_SESSION', 'LONG_LIVED_TOKEN');

ALTER TABLE user_sessions ADD COLUMN created_at TIMESTAMP DEFAULT NULL;
ALTER TABLE user_sessions ADD COLUMN token_type token_type NOT NULL DEFAULT 'USER_SESSION';
ALTER TABLE user_sessions ADD COLUMN revoked boolean DEFAULT false NOT NULL;
ALTER TABLE user_sessions ADD COLUMN description TEXT DEFAULT NULL;

-- Add a conditional unique index to enforce 1:1 relationship for active long-lived tokens
CREATE UNIQUE INDEX ON public.user_sessions (user_id)
WHERE token_type = 'LONG_LIVED_TOKEN' AND revoked = false;
