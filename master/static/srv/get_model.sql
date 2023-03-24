SELECT m.id, m.name, m.description, m.notes, m.metadata, m.creation_time, m.last_updated_time, array_to_json(m.labels) AS labels, m.user_id, u.username, m.workspace_id, m.archived, COUNT(mv.version) as num_versions
FROM models as m
  LEFT JOIN model_versions as mv
    ON mv.model_id = m.id
  LEFT JOIN users as u ON u.id = m.user_id
WHERE m.name = $1
GROUP BY m.id, u.id;
