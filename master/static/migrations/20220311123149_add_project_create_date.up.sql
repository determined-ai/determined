ALTER TABLE projects
ADD COLUMN created_at timestamp with time zone NOT NULL DEFAULT NOW();
