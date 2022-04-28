WITH x AS (
  DELETE FROM user_profile_images
  WHERE user_id = $1
)
INSERT INTO user_profile_images (user_id, file_data)
VALUES ($1, $2::bytea)
RETURNING id;
