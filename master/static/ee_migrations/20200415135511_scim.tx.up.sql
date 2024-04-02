CREATE SCHEMA scim;
CREATE TABLE scim.users (
  id          uuid PRIMARY KEY NOT NULL,
  user_id     INTEGER NOT NULL REFERENCES users(id),
  external_id TEXT NULL,
  name        jsonb NOT NULL,
  emails      jsonb NOT NULL
);
