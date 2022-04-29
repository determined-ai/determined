-- Handle this in server code because SQL was silently blocking delete.
DROP TRIGGER autoupdate_user_image_deleted on user_profile_images;

-- Allow upsert
ALTER TABLE user_profile_images ADD CONSTRAINT profile_per_user UNIQUE (user_id);
