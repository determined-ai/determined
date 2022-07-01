CREATE TABLE user_web_settings (
	user_id integer REFERENCES public.users(id) ON DELETE CASCADE NOT NULL,
	key text NOT NULL,
	storage_path text NOT NULL,
	value jsonb,
    CONSTRAINT user_web_settings_uniq UNIQUE (user_id, key, storage_path)
);