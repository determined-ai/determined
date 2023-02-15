ALTER TABLE public.allocations
	DROP COLUMN agent_label;
DELETE FROM resource_aggregates WHERE aggregation_type = 'agent_label';
