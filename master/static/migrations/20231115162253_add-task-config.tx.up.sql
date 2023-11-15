ALTER TABLE tasks
ADD config jsonb NOT NULL DEFAULT('{}');