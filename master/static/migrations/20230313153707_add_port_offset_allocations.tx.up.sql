ALTER TABLE allocations
ADD ports jsonb DEFAULT '{}' not null;
