-- Returns pairs of metric names and trial_ids and if they are numeric or not.
WITH trial_metrics as (
SELECT
	name,
	trial_id,
	CASE safe_sum(entries)
		WHEN safe_sum(entries) FILTER (WHERE metric_type = 'number') THEN 'number'
		WHEN safe_sum(entries) FILTER (WHERE metric_type = 'string') THEN 'string'
		WHEN safe_sum(entries) FILTER (WHERE metric_type = 'date') THEN 'date'
		WHEN safe_sum(entries) FILTER (WHERE metric_type = 'object') THEN 'object'
		WHEN safe_sum(entries) FILTER (WHERE metric_type = 'boolean') THEN 'boolean'
		WHEN safe_sum(entries) FILTER (WHERE metric_type = 'array') THEN 'array'
		WHEN safe_sum(entries) FILTER (WHERE metric_type = 'null') THEN 'null'
		ELSE 'string'
	END as metric_type
FROM (
	SELECT
	name,
	CASE
		WHEN jsonb_typeof(metrics->$2->name) = 'string' THEN
			CASE
				WHEN (metrics->$2->name)::text = '"Infinity"'::text THEN 'number'
				WHEN (metrics->$2->name)::text = '"-Infinity"'::text THEN 'number'
				WHEN (metrics->$2->name)::text = '"NaN"'::text THEN 'number'
				WHEN metrics->$2->>name ~
					'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$' THEN 'date'
				ELSE 'string'
			END
		ELSE jsonb_typeof(metrics->$2->name)
	END as metric_type,
	trial_id,
	count(1) as entries
	FROM (
		SELECT DISTINCT
		jsonb_object_keys(s.metrics->$2) as name
		FROM metrics s
		-- PERF: we can do a fancier check to avoid checking custom_type when partition is not generic.
		WHERE s.trial_id = $1 AND partition_type = $3 AND custom_type = $4 AND not archived
	) names, metrics
	JOIN trials ON trial_id = trials.id
	WHERE trials.id = $1 AND metrics.partition_type = $3 AND metrics.custom_type = $4 AND not metrics.archived
	GROUP BY name, metric_type, trial_id
) typed
where metric_type IS NOT NULL
GROUP BY name, trial_id
ORDER BY trial_id, name
),
-- Filters to only numeric metrics.
numeric_trial_metrics as (
SELECT name, trial_id
FROM trial_metrics
WHERE metric_type = 'number'
),
-- Calculates count, sum, min, max on each numeric metric name and trial ID pair.
-- Also adds just the name for non numeric metrics to ensure we record every metric.
trial_metric_aggs as (
SELECT
	name,
	ntm.trial_id,
	count(1) as count_agg,
	safe_sum((metrics.metrics->$2->>name)::double precision) as sum_agg,
	min((metrics.metrics->$2->>name)::double precision) as min_agg,
	max((metrics.metrics->$2->>name)::double precision) as max_agg,
	'number' as metric_type
FROM numeric_trial_metrics ntm INNER JOIN metrics
ON metrics.trial_id=ntm.trial_id
WHERE metrics.metrics->$2->name IS NOT NULL AND metrics.partition_type = $3 AND metrics.custom_type = $4 AND not metrics.archived
GROUP BY 1, 2
UNION
SELECT
	name,
	trial_id,
	NULL as count_agg,
	NULL as sum,
	NULL as min,
	NULL as max,
	metric_type as metric_type
FROM trial_metrics
WHERE metric_type != 'number'
),
-- Gets the last reported metric for each trial. Note if we report
-- {"a": 1} and {"b": 1} we consider {"b": 1} to be the last reported
-- metric and "a"'s last will be NULL.
latest_metrics as (
  SELECT s.trial_id,
	unpacked.key as name,
	unpacked.value as latest_value
  FROM (
	  SELECT s.*,
		ROW_NUMBER() OVER(
		  PARTITION BY s.trial_id
		  ORDER BY s.end_time DESC
		) as rank
	  FROM metrics s
	  JOIN trials ON s.trial_id = trials.id
	  WHERE s.trial_id = $1 AND partition_type = $3 AND custom_type = $4 AND not archived
	) s, jsonb_each(s.metrics->$2) unpacked
  WHERE s.rank = 1
),
-- Adds the last reported metric to the aggregation.
combined_latest_agg as (SELECT
	coalesce(lt.trial_id, tma.trial_id) as trial_id,
	coalesce(lt.name, tma.name) as name,
	tma.count_agg,
	tma.sum_agg,
	tma.min_agg,
	tma.max_agg,
	lt.latest_value,
	tma.metric_type
FROM latest_metrics lt FULL OUTER JOIN trial_metric_aggs tma ON
	lt.trial_id = tma.trial_id AND lt.name = tma.name
) SELECT name, jsonb_build_object(
    'count', count_agg,
    'sum', sum_agg,
    'min', CASE WHEN max_agg = 'NaN'::double precision
        THEN 'NaN'::double precision ELSE min_agg END,
    'max', max_agg,
    'last', latest_value,
    'type', metric_type
) FROM combined_latest_agg;
