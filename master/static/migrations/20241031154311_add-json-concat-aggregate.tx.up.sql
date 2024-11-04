create aggregate jsonb_concat_agg(jsonb)(
	sfunc = jsonb_concat(jsonb, jsonb),
	stype = jsonb
);
