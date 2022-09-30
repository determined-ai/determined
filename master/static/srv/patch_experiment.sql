UPDATE experiments e
SET config = config || $2, notes = $3, group_id = $4
WHERE e.id = $1
RETURNING e.id
