ALTER TABLE public.allocation_sessions
	DROP COLUMN owner_id int REFERENCES users(id);
