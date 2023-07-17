UPDATE generic_metrics
SET custom_type = replace(custom_type, '.', '_')
WHERE custom_type LIKE '%.%';
