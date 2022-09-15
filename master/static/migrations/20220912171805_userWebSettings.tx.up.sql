CREATE TABLE IF NOT EXISTS user_settings_webs (
  user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  value jsonb,
  CONSTRAINT user_setting_web_uniq UNIQUE (user_id)
);

-- CREATE UNIQUE INDEX user_web_setting_uniq ON user_web_settings(user_id int4_ops,key text_ops,storage_path text_ops);
