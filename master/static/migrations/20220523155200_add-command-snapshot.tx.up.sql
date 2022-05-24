CREATE TABLE command_state (
	task_id text PRIMARY KEY REFERENCES tasks(task_id) ON DELETE CASCADE NOT NULL,
	registered_time timestamp with time zone NOT NULL DEFAULT NOW(),
	allocation_id text REFERENCES allocations(allocation_id) ON DELETE CASCADE NOT NULL,
	generic_command_spec jsonb
);

CREATE INDEX ix_command_state_allocation_id ON command_state USING btree (allocation_id);
