UPDATE allocations a
SET end_time = start_time
WHERE start_time > end_time
