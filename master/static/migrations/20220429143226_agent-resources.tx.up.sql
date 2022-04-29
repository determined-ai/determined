CREATE TABLE allocation_resources (
	resource_id text PRIMARY KEY,
	allocation_id text REFERENCES allocations(allocation_id) ON DELETE CASCADE NOT NULL,
	rank int,
	started jsonb,
	exited jsonb,
	daemon boolean
);

CREATE TABLE resourcemanagers_agent_containers (
	container_id text PRIMARY KEY,
	resource_id text REFERENCES allocation_resources(resource_id) ON DELETE CASCADE NOT NULL,
	agent_id text NOT NULL,
	state text,
	devices jsonb
);
