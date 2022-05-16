CREATE TABLE resourcemanagers_agent_agentstate (
	id serial PRIMARY KEY,
	agent_id text UNIQUE NOT NULL,
	uuid text UNIQUE NOT NULL,
	resource_pool_name text NOT NULL,
	label text NOT NULL,
	user_enabled boolean,
	user_draining boolean,
	max_zero_slot_containers integer NOT NULL,
	slots jsonb,
	containers jsonb
);
