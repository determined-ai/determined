WITH sorted_labels AS (
  SELECT label
  FROM (
    SELECT id, UNNEST(labels) AS label
    FROM model_versions
  ) all_labels
  GROUP BY label
  ORDER BY COUNT(DISTINCT(id)) DESC, label ASC
)
SELECT array_to_json(ARRAY_AGG(label)) AS labels
FROM sorted_labels;
