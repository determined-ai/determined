UPDATE trials SET total_batches = sub.new_max_total_batches_processed
FROM (
	SELECT max(q.total_batches) AS new_max_total_batches_processed, trial_id
	FROM (
		SELECT coalesce(max(s.total_batches), 0) as total_batches, s.trial_id
		FROM steps s
		GROUP BY s.trial_id
		UNION ALL
		SELECT coalesce(max(v.total_batches), 0) AS total_batches, v.trial_id
		FROM validations v
		GROUP BY v.trial_id
	) q
	GROUP BY trial_id
) AS sub
WHERE id = sub.trial_id;
