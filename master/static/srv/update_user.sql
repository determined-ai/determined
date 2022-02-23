UPDATE users SET display_name = $2, password_hash = $3
WHERE username = $1
RETURNING username, display_name
