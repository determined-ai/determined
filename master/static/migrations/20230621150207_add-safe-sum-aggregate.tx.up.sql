CREATE OR REPLACE FUNCTION safe_sum_accumulate(float8, float8, OUT float8)
  RETURNS float8 AS $$
  BEGIN
    -- Check for potential overflow
    BEGIN
      IF $1 IS NULL THEN
        $3 := $2;
      ELSIF $2 IS NULL THEN
        $3 := $1;
      ELSE
        $3 := $1 + $2;
      END IF;
    EXCEPTION
      WHEN numeric_value_out_of_range THEN
        IF $1 < 0 THEN
          $3 := '-Infinity';
        ELSE
          $3 := 'Infinity';
        END IF;
    END;
  END;
$$ LANGUAGE plpgsql;

DROP AGGREGATE IF EXISTS safe_sum(float8);

CREATE AGGREGATE safe_sum(float8) (
  SFUNC = safe_sum_accumulate,
  STYPE = float8
);
