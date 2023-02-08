SELECT m.id, m.name, m.description, m.notes, m.metadata, m.creation_time, m.last_updated_time, array_to_json(m.labels) AS labels, u.username, m.user_id, m.workspace_id, m.archived, COUNT(mv.version) as num_versions FROM models as m
LEFT JOIN model_versions as mv ON mv.model_id = m.id
LEFT JOIN users as u ON u.id = m.user_id
WHERE ($1 = 0 OR m.id = $1)
AND ($2 = '' OR m.archived = $2::BOOL)
AND ($3 = '' OR (u.username IN (SELECT unnest(string_to_array($3, ',')))))
AND ($4 = '' OR m.user_id IN (SELECT unnest(string_to_array($4, ',')::int [])))
AND ($5 = '' OR (m.labels && string_to_array($5, ',')))
AND ($6 = '' OR m.name ILIKE $6)
AND ($7 = '' OR m.description ILIKE $7)
AND ($8 = 0 or m.workspace_id = $8::int)
AND ($9 = '' OR m.workspace_id IN (SELECT unnest(string_to_array($9, ',')::int [])))
GROUP BY m.id, u.id
ORDER BY %s;
