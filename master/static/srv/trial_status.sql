SELECT t.state AS State
FROM trials t
WHERE t.id = $1
