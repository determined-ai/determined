DROP TYPE IF EXISTS token_type CASCADE;
ALTER TABLE user_sessions DROP COLUMN created_at;
-- ALTER TABLE user_sessions DROP COLUMN token_type;
ALTER TABLE user_sessions DROP COLUMN revoked;
ALTER TABLE user_sessions DROP COLUMN description;

CREATE TYPE token_type AS ENUM ('USER_SESSION', 'ACCESS_TOKEN');

ALTER TABLE user_sessions ADD COLUMN created_at TIMESTAMP DEFAULT NULL;
ALTER TABLE user_sessions ADD COLUMN token_type token_type NOT NULL DEFAULT 'USER_SESSION';
ALTER TABLE user_sessions ADD COLUMN revoked boolean DEFAULT false NOT NULL;
ALTER TABLE user_sessions ADD COLUMN description TEXT DEFAULT NULL;
