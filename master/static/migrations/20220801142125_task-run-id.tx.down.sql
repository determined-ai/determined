-- Since the migration was destructive, it cannot truly be undone.
ALTER TABLE public.trials
ADD COLUMN run_id integer NOT NULL DEFAULT 0;
