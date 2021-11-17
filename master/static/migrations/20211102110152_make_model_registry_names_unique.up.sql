WITH m AS (
  SELECT name, COUNT(*) AS count FROM models GROUP BY name
),
names AS (
  SELECT name FROM m WHERE count > 1
)
UPDATE models SET name = CONCAT(name, id::text)
WHERE name IN (SELECT name FROM names);

ALTER TABLE public.models ADD CONSTRAINT models_name_unique UNIQUE (name);
