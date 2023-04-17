CREATE TABLE trial_profiler_metrics (
    id BIGSERIAL,
    values FLOAT4 [] NOT NULL,
    batches INT [] NOT NULL,
    ts TIMESTAMP WITH TIME ZONE [] NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::JSONB
);

CREATE INDEX trial_profiler_metric_labels ON public.trial_profiler_metrics USING gin (
    labels
);
