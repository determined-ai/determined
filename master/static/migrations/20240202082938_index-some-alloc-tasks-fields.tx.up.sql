CREATE INDEX ix_allocations_task_id ON allocations USING btree (task_id);

CREATE INDEX ix_tasks_job_id ON tasks USING btree (job_id);
