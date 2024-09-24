CREATE TYPE token_type AS ENUM ('USER_SESSION', 'ACCESS_TOKEN');

ALTER TABLE user_sessions ADD COLUMN created_at TIMESTAMP DEFAULT NULL;
ALTER TABLE user_sessions ADD COLUMN token_type token_type NOT NULL DEFAULT 'USER_SESSION';
ALTER TABLE user_sessions ADD COLUMN revoked boolean DEFAULT false NOT NULL;
ALTER TABLE user_sessions ADD COLUMN description TEXT DEFAULT NULL;

-- Add a conditional unique index to enforce 1:1 relationship for active access tokens
CREATE UNIQUE INDEX ON public.user_sessions (user_id)
WHERE token_type = 'ACCESS_TOKEN' AND revoked = false;
