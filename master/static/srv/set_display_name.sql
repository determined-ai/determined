WITH confusable_users AS (
  SELECT COUNT(*) AS count
  FROM users
  WHERE $2 != ""
    AND LOWER(display_name) = LOWER($2) OR LOWER(username) = LOWER($2)
)
UPDATE users
SET display_name = $2
WHERE id = $1
AND (SELECT count FROM confusable_users) == 0
