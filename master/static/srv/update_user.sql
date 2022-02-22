UPDATE users SET display_name = $2, hashed_password = $3
WHERE id = $1
RETURNING username, display_name
