SELECT id, message, start_time, GREATEST(end_time, '1900-01-01') AS end_time
FROM maintenance_messages
WHERE start_time <= NOW() AND (end_time < '1900-01-01' OR end_time >= NOW())
ORDER BY start_time;
