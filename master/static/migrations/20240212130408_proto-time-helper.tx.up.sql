CREATE OR REPLACE FUNCTION public.proto_time(ts timestamptz)
    RETURNS json
    LANGUAGE sql IMMUTABLE
    RETURNS NULL ON NULL INPUT
AS $$
    SELECT json_build_object(
        -- Seconds since epoch
        'seconds',  floor(extract(epoch FROM ts))::bigint, 
        -- Fractional part in nanos since epoch
        'nanos',    (MOD(extract(milliseconds FROM ts)::decimal, 1000::decimal)*1000000)::int
    )
$$;
