SELECT file_data AS photo
FROM users u
LEFT JOIN user_profile_images img ON u.id = img.user_id
WHERE u.username = $1
LIMIT 1;
