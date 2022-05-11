WITH up AS (
  DELETE FROM user_profile_images
  WHERE user_id = $1
  RETURNING user_id
)
UPDATE users SET modified_at = NOW() WHERE id IN (
  SELECT user_id FROM up
);
