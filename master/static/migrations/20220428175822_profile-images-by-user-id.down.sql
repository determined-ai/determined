CREATE TRIGGER autoupdate_user_image_deleted
BEFORE DELETE ON user_profile_images
FOR EACH ROW EXECUTE PROCEDURE autoupdate_user_image_deleted();

ALTER TABLE user_profile_images DROP CONSTRAINT profile_per_user;
