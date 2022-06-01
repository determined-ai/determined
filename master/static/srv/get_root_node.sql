WITH c AS (
  SELECT id,
    config->>'name' AS name,
    lineage AS children,
    parent_id
  FROM experiments
  WHERE lineage[1] = $1 OR lineage[2] = $1
)
SELECT
  experiments.id,
  experiments.config->>'name' AS name,
  (SELECT json_agg(x.fmted)
    FROM (
        (SELECT
            json_build_object(
                'id', c.id,
                'name', c.name
            ) as fmted
        FROM c
        GROUP by c.id, c.name)
    ) x
  ) AS children
FROM experiments
JOIN c ON c.parent_id = experiments.id
WHERE experiments.id = $1
GROUP BY experiments.id, c.*
