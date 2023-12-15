SELECT id, message, start_time, end_time
FROM maintenance_messages
WHERE start_time <= NOW() AND end_time >= NOW()
ORDER BY start_time;
