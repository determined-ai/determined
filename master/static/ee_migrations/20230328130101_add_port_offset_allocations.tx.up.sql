ALTER TABLE allocations
ADD COLUMN IF NOT EXISTS ports jsonb DEFAULT '{}' not null;
