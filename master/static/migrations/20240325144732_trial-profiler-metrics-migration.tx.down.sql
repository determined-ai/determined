/*
 Rollback the data migration of inserting historical system metrics data from `trial_profiler_metrics`
 to `metrics`.
 */

-- Assume that everything in `system_metrics` that is also in `trial_profiler_metrics` should be deleted.
delete from system_metrics sm
    using trial_profiler_metrics tpm
where (tpm.labels->>'trialId')::int=sm.trial_id
;
