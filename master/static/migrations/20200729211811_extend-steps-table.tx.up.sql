ALTER TABLE public.steps ADD COLUMN prior_batches_processed integer NULL;

WITH legacy_num_batches AS (
    SELECT
        id experiment_id,
        (e.config->>'batches_per_step')::int num_batches
    FROM public.experiments e
)
-- first backfill num_batches for each step with the value of batches_per_step, if missing.
UPDATE public.steps AS s
SET num_batches = (
    SELECT num_batches
    FROM legacy_num_batches b
    JOIN public.trials t ON t.experiment_id = b.experiment_id
    WHERE t.id = s.trial_id
)
WHERE s.num_batches IS NULL;

-- then backfill prior_batches_processed using the value of num_batches we just backfilled.
UPDATE public.steps AS s
SET prior_batches_processed = (
    SELECT coalesce(sum(ss.num_batches), 0)
    FROM public.steps ss
    WHERE ss.id < s.id AND ss.trial_id = s.trial_id
)
WHERE s.prior_batches_processed IS NULL;
