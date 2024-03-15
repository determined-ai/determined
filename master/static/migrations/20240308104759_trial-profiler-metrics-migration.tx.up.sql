/*
  Migrates existing profiling metrics (in `trial_profiler_metrics`) to `metrics`.

  Queries `trial_profiler_metrics` to downsample and shim all system metrics data to fit
  generic metrics `metrics` schema.
 */

insert into system_metrics (trial_id, end_time, metrics, trial_run_id, partition_type, metric_group)
-- Long-winded query to parse and massage existing data in `trial_profiler_metrics` to fit the new schema
select
    pm.trial_id as trial_id,
    -- Use latest time of aggregate GPU groups.
    max(pm.ts) as end_time,
    (
        case pm.mgroup
            when 'gpu' then jsonb_build_object(
                pm.agent_id,
                jsonb_object_agg(
                    -- GPU UUID should never be null here, but SQL doesn't know that.
                    coalesce(pm.gpu_uuid, ''), pm.metrics
                )
            )
            else jsonb_object_agg(pm.agent_id, pm.metrics)
        end
    ) as metrics,
    coalesce(t.run_id, 1) as trial_run_id,
    'PROFILING' as partition_type,
    pm.mgroup as metric_group
from
    (
        -- Query to downsample existing metrics to one per metric name <> group <> agent <> trial <> gpu per second
        select
            jsonb_object_agg(
                -- Change the `free_memory` metric name to `mem_free`.
                case
                    when tpm.labels->>'name'='free_memory' then 'mem_free'
                    else tpm.labels->>'name'
                end,
                v
            ) as metrics,
            date_trunc('second', t) as tsec,
            -- Use most recent timestamp of this trial <> agent <> gpu group for the 'new' batch
            max(t) as ts,
            (
                case
                    when tpm.labels->>'name' like 'gpu\_%' then 'gpu'
                    when tpm.labels->>'name' like 'net\_%' then 'network'
                    when tpm.labels->>'name' like 'disk\_%' then 'disk'
                    when tpm.labels->>'name' like 'cpu\_%' then 'cpu'
                    when tpm.labels->>'name'='free_memory' then 'memory'
                    end
            ) as mgroup,
            (tpm.labels->>'trialId')::int as trial_id,
            tpm.labels->>'agentId' as agent_id,
            tpm.labels->>'gpuUuid' as gpu_uuid
        from trial_profiler_metrics tpm
            -- Only get every 10 reported (time, value) pairs (currently reports every 0.1s).
            cross join lateral unnest(tpm.values, tpm.ts) with ordinality as vals(v, t, n)
        where
            tpm.labels->>'metricType' = 'PROFILER_METRIC_TYPE_SYSTEM'
            and vals.n % 10 = 1
        group by trial_id, mgroup, tsec, agent_id, gpu_uuid
    ) as pm
left join trials t on t.id=pm.trial_id
group by pm.trial_id, t.run_id, pm.mgroup, pm.tsec, pm.agent_id
;
