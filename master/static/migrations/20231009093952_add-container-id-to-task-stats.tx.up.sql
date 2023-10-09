ALTER TABLE task_stats
  ADD COLUMN container_id TEXT;

CREATE INDEX idx_task_stats_container_id_id ON task_stats USING btree (container_id)
  WHERE container_id IS NOT NULL;
