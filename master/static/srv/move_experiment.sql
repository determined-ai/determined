UPDATE experiments SET project_id = $2
WHERE id = $1
RETURNING id;
