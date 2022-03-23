CREATE FUNCTION public.frequencies(vals anyarray) RETURNS jsonb
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT coalesce(jsonb_agg(row_to_json(counts)), '[]'::jsonb)
    FROM (
        SELECT to_jsonb(unnest) as value, count(*)
        FROM unnest(vals) GROUP BY unnest
    ) counts
$$;
