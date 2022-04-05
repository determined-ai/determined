SELECT *
FROM proto_checkpoints_view c
WHERE c.training->>'trial_id' = $1
ORDER BY c.report_time DESC
