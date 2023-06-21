DROP AGGREGATE IF EXISTS safe_sum(float8);

DROP FUNCTION IF EXISTS safe_sum_accumulate(float8, float8, OUT float8);
