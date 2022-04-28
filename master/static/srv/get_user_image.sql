SELECT file_data AS photo
FROM user_profile_images
WHERE user_id = $1::int
LIMIT 1;
