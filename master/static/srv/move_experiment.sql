UPDATE experiments SET project_id = $2, group_id = NULL
WHERE id = $1
RETURNING id;
