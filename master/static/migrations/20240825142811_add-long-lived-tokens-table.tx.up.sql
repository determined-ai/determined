CREATE TABLE public.long_lived_tokens (
    id integer NOT NULL,
    user_id integer NOT NULL,
    token_value_hash BYTEA NOT NULL, -- Hash of the token value for secure storage
    expiration_timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    PRIMARY KEY(id) ,
    FOREIGN KEY(user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

-- Adding an index on token_value_hash for faster lookups
CREATE INDEX idx_token_value_hash ON long_lived_tokens(token_value_hash);