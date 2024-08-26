-- CREATE TABLE public.long_lived_tokens (
--     id integer NOT NULL,
--     user_id integer NOT NULL,
--     token_value_hash BYTEA NOT NULL, -- Hash of the token value for secure storage
--     expiration_timestamp TIMESTAMP NOT NULL,
--     created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, 
--     PRIMARY KEY(id) ,
--     FOREIGN KEY(user_id) REFERENCES public.users(id) ON DELETE CASCADE
-- );
-- CRUD Functions

-- Create (Insert)
INSERT INTO long_lived_tokens (id, user_id, token_value_hash, expiration_timestamp)
VALUES (1, 1, 'hashed_token_value', '2024-12-31 23:59:59+00');

CREATE OR REPLACE FUNCTION create_long_lived_token(
    p_user_id UUID,
    p_token_value_hash BYTEA,
    p_expiration_timestamp TIMESTAMP
) RETURNS UUID AS $$
DECLARE
    v_id UUID;
BEGIN
    INSERT INTO long_lived_tokens (user_id, token_value_hash, expiration_timestamp)
    VALUES (p_user_id, p_token_value_hash, p_expiration_timestamp)
    RETURNING id INTO v_id;

    RETURN v_id;
END;
$$ LANGUAGE plpgsql;

-- Read (Select by id)
SELECT * FROM long_lived_tokens
WHERE token_value_hash = 'hashed_token_value';

CREATE OR REPLACE FUNCTION get_long_lived_token_by_id(
    p_id UUID
) RETURNS TABLE(
    id UUID,
    user_id UUID,
    token_value_hash BYTEA,
    created_at TIMESTAMP,
    expiration_timestamp TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY 
    SELECT * 
    FROM long_lived_tokens
    WHERE id = p_id;
END;
$$ LANGUAGE plpgsql;

-- Read (Select by token_value_hash)
CREATE OR REPLACE FUNCTION get_long_lived_token_by_hash(
    p_token_value_hash BYTEA
) RETURNS TABLE(
    id UUID,
    user_id UUID,
    token_value_hash BYTEA,
    created_at TIMESTAMP,
    expiration_timestamp TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY 
    SELECT * 
    FROM long_lived_tokens
    WHERE token_value_hash = p_token_value_hash;
END;
$$ LANGUAGE plpgsql;

-- Update (We'll assume we might want to update the expiration)
UPDATE long_lived_tokens
SET expiration_timestamp = '2025-01-31 23:59:59+00'
WHERE token_value_hash = 'hashed_token_value';

CREATE OR REPLACE FUNCTION update_long_lived_token_expiry(
    p_id UUID,
    p_new_expiration_timestamp TIMESTAMP
) RETURNS VOID AS $$
BEGIN
    UPDATE long_lived_tokens
    SET expiration_timestamp = p_new_expiration_timestamp
    WHERE id = p_id;
END;
$$ LANGUAGE plpgsql;

-- Delete
DELETE FROM long_lived_tokens
WHERE id = 1;

CREATE OR REPLACE FUNCTION delete_long_lived_token(
    p_id UUID
) RETURNS VOID AS $$
BEGIN
    DELETE FROM long_lived_tokens
    WHERE id = p_id;
END;
$$ LANGUAGE plpgsql;
