CREATE OR REPLACE FUNCTION public.proto_time(ts timestamptz)
    RETURNS json
    LANGUAGE sql IMMUTABLE
    RETURNS NULL ON NULL INPUT
AS $$
    SELECT json_build_object(
        -- Seconds since epoch
        'seconds',  floor(extract(EPOCH from ts))::BIGINT, 
        -- Fractional part in nanos since epoch
        'nanos',    ((extract(EPOCH from ts) - floor(extract(EPOCH from ts))) * 1000000000)::INT
    )
$$;
