INSERT INTO user_profile_images (user_id, file_data)
VALUES ($1, $2::bytea)
ON CONFLICT (user_id)
DO
   UPDATE SET file_data = $2::bytea
RETURNING id;
