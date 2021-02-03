CREATE OR REPLACE FUNCTION public.page_info(total bigint, "offset" int, "limit" int)
    RETURNS json
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
