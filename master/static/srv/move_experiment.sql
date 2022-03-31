UPDATE experiments SET project_id = $2
WHERE experiments.id = $1
RETURNING experiments.id;
