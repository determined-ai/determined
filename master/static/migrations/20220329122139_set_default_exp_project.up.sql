-- Set default project (usually 1, but make sure)
DO $$
BEGIN
EXECUTE format('ALTER TABLE experiments ALTER COLUMN project_id SET DEFAULT %L'
             , (SELECT MIN(id) FROM projects WHERE name = 'Uncategorized'));
END $$;

-- Update any experiments after the previous release
UPDATE experiments SET project_id = (
  SELECT MIN(id) FROM projects WHERE name = 'Uncategorized'
)
WHERE project_id IS NULL;

-- Disallow nulls in the future
ALTER TABLE experiments ALTER project_id SET NOT NULL;
