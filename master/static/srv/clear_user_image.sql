DELETE FROM user_profile_images
WHERE user_id = $1;

UPDATE users SET modified_at = NOW() WHERE user_id = $1;
