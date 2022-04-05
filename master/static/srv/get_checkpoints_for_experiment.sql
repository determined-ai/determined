SELECT *
FROM proto_checkpoints_view c
WHERE CAST(c.training->>'experiment_id' AS integer) = $1
ORDER BY c.report_time DESC
