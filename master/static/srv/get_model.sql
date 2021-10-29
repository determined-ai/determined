SELECT m.id, m.name, m.description, m.metadata, m.creation_time, m.last_updated_time, array_to_json(m.labels) AS labels, m.readme, u.username, m.archived, COUNT(mv.version) as num_versions
FROM models as m
  LEFT JOIN model_versions as mv
    ON mv.model_id = m.id
  LEFT JOIN users as u ON u.id = m.user_id
WHERE m.id = $1
GROUP BY m.id, u.id;
