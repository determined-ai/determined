CREATE TABLE user_profile_images (
  id SERIAL PRIMARY KEY,
  user_id INT,
  file_data BYTEA NOT NULL,
  CONSTRAINT photo_for_user
   FOREIGN KEY(user_id)
      REFERENCES users(id)
   ON DELETE CASCADE
);

ALTER TABLE users ADD COLUMN modified_at TIMESTAMP DEFAULT current_timestamp;

CREATE OR REPLACE FUNCTION autoupdate_user_image_modified() RETURNS trigger AS $$
BEGIN
  UPDATE users SET modified_at = NOW() WHERE users.id = NEW.user_id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_user_image_modified
BEFORE INSERT OR UPDATE ON user_profile_images
FOR EACH ROW EXECUTE PROCEDURE autoupdate_user_image_modified();

CREATE OR REPLACE FUNCTION autoupdate_user_image_deleted() RETURNS trigger AS $$
BEGIN
  UPDATE users SET modified_at = NOW() WHERE users.id = OLD.user_id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_user_image_deleted
BEFORE DELETE ON user_profile_images
FOR EACH ROW EXECUTE PROCEDURE autoupdate_user_image_deleted();
