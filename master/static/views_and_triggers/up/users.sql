CREATE FUNCTION autoupdate_user_image_deleted() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  UPDATE users SET modified_at = NOW() WHERE users.id = OLD.user_id;
  RETURN NEW;
END;
$$;
CREATE TRIGGER autoupdate_user_image_deleted BEFORE DELETE ON user_profile_images FOR EACH ROW EXECUTE PROCEDURE autoupdate_user_image_deleted();

CREATE FUNCTION autoupdate_user_image_modified() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  UPDATE users SET modified_at = NOW() WHERE users.id = NEW.user_id;
  RETURN NEW;
END;
$$;
CREATE TRIGGER autoupdate_user_image_modified BEFORE INSERT OR UPDATE ON user_profile_images FOR EACH ROW EXECUTE PROCEDURE autoupdate_user_image_modified();

CREATE FUNCTION set_modified_time() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.modified_at := now();
    RETURN NEW;
END;
$$;
CREATE TRIGGER autoupdate_users_modified_at BEFORE INSERT OR UPDATE OF username, password_hash, admin, active, display_name, remote ON users FOR EACH ROW EXECUTE PROCEDURE set_modified_time();
