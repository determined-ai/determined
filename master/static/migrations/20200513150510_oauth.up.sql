CREATE SCHEMA oauth;
CREATE TABLE oauth.tokens (
  access text NOT NULL,
  access_create_at timestamptz NOT NULL,
  access_expires_in bigint NOT NULL,
  client_id text NOT NULL,
  code text NOT NULL,
  code_create_at timestamptz NOT NULL,
  code_expires_in bigint NOT NULL,
  redirect_uri text NOT NULL,
  refresh text NOT NULL,
  refresh_create_at timestamptz NOT NULL,
  refresh_expires_in bigint NOT NULL,
  scope text NOT NULL,
  user_id text NOT NULL,

  id bigserial,

  CONSTRAINT oauth_tokens_pkey PRIMARY KEY (id)
);

CREATE INDEX idx_oauth_tokens_code ON oauth.tokens (code);
CREATE INDEX idx_oauth_tokens_access ON oauth.tokens (access);
CREATE INDEX idx_oauth_tokens_refresh ON oauth.tokens (refresh);

CREATE TABLE oauth.clients (
  id text NOT NULL,
  secret text NOT NULL,
  domain text NOT NULL,
  name text NOT NULL,

  CONSTRAINT oauth_clients_pkey PRIMARY KEY (id)
);
