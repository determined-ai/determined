CREATE TABLE trial_profiler_metrics (
    values  float4[]    NOT NULL,
    batches int[]       NOT NULL,
    ts      timestamp[] NOT NULL,
    labels  jsonb       NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX trial_profiler_metric_labels ON public.trial_profiler_metrics USING gin (labels);