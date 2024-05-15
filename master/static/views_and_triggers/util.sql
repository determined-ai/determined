CREATE FUNCTION determined_code.page_info(total bigint, "offset" integer, "limit" integer) RETURNS json
    LANGUAGE sql IMMUTABLE
    AS $$
WITH start_index AS (
    SELECT (CASE WHEN "offset" < -total OR "offset" > total THEN total
                 WHEN "offset" < 0 THEN total + "offset"
                 ELSE "offset" END) AS start_index
), end_index AS (
    SELECT (CASE WHEN "limit" = -2 THEN (SELECT start_index FROM start_index)
                 WHEN "limit" = -1 THEN total
                 WHEN "limit" = 0 THEN least(100::bigint + (SELECT start_index FROM start_index), total)
                 WHEN (SELECT start_index FROM start_index) + "limit" > total THEN total
                 ELSE (SELECT start_index FROM start_index) + "limit" END) AS end_index
), page_info AS (
    SELECT
        total AS total,
        "offset" AS "offset",
        "limit" AS "limit",
        (SELECT start_index FROM start_index) AS start_index,
        (SELECT end_index FROM end_index) AS end_index)
SELECT row_to_json(p)
FROM page_info p
$$;

CREATE FUNCTION determined_code.proto_time(ts timestamp with time zone) RETURNS json
    LANGUAGE sql IMMUTABLE STRICT
    AS $$
    SELECT json_build_object(
        -- Seconds since epoch
        'seconds',  floor(extract(epoch FROM ts))::bigint,
        -- Fractional part in nanos since epoch
        'nanos',    (MOD(extract(milliseconds FROM ts)::decimal, 1000::decimal)*1000000)::int
    )
$$;

CREATE FUNCTION determined_code.retention_timestamp() RETURNS timestamp with time zone
    LANGUAGE plpgsql
    AS $$
    BEGIN
        RETURN transaction_timestamp();
    END
    $$;

CREATE FUNCTION determined_code.try_float8_cast(text) RETURNS double precision
    LANGUAGE sql IMMUTABLE STRICT
    AS $_$
            SELECT
                CASE
                    WHEN $1 ~ e'^-?(?:0|[1-9]\\d*)'
                               '(?:\\.\\d+)?(?:[eE][+-]?\\d+)?$' THEN
                        $1::float8
                END;
        $_$;

CREATE AGGREGATE determined_code.jsonb_collect(jsonb) (
    SFUNC = jsonb_concat,
    STYPE = jsonb,
    INITCOND = '{}'
);
