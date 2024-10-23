ALTER table public.tasks
    ADD COLUMN log_policy_matched text;

ALTER TABLE public.runs 
    RENAME COLUMN log_signal TO log_policy_matched;