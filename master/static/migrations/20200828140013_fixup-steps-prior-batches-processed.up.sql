-- This backfills steps.prior_batches_processed, again.
-- In 0.13.0, a bad migration was released that set this value incorrectly, but that was also
-- so inefficient it hung for large installations. If a user skipped 0.13.0, this will
-- be a no-op since migration 20200729211811 was corrected post-release; however, if a user
-- already successfully upgraded to 0.13.1, this will correct the old, incorrect values
-- from migration found in 0.13.0.
UPDATE public.steps AS s
SET prior_batches_processed = (
    SELECT coalesce(sum(ss.num_batches), 0)
    FROM public.steps ss
    WHERE ss.id < s.id AND ss.trial_id = s.trial_id
);
