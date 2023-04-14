ALTER TABLE public.allocation_sessions
ADD COLUMN owner_id int REFERENCES users(id);

-- Add owner_id for in progress trials.
UPDATE public.allocation_sessions allocation_sessions
SET owner_id = experiments.owner_id
FROM public.experiments AS experiments
INNER JOIN public.trials AS trials ON experiments.id = trials.experiment_id
INNER JOIN
    public.allocations AS allocations
    ON trials.task_id = allocations.task_id
WHERE allocation_sessions.allocation_id = allocations.allocation_id;
