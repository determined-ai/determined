UPDATE experiments e
SET config = config || $2,
    framework = $3
WHERE e.id = $1
RETURNING e.id
