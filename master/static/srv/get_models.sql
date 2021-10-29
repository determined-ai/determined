SELECT m.id, m.name, m.description, m.metadata, m.creation_time, m.last_updated_time, array_to_json(m.labels) AS labels, m.readme, u.username, m.archived, COUNT(mv.version) as num_versions FROM models as m
LEFT JOIN model_versions as mv ON mv.model_id = m.id
LEFT JOIN users as u ON u.id = m.user_id
WHERE ($1 = '' OR m.archived = $1::BOOL)
AND ($2 = '' OR (u.username IN (SELECT unnest(string_to_array($2, ',')))))
AND ($3 = '' OR (m.labels <@ string_to_array($3, ',')))
AND ($4 = '' OR LOWER(m.name) LIKE $4)
AND ($5 = '' OR LOWER(m.description) LIKE $5)
GROUP BY m.id, u.id
ORDER BY %s;
