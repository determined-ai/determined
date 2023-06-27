CREATE TABLE cert_and_key_info (
    serial_number bigint unique NOT NULL,
    cert bytea NOT NULL,
    key bytea NOT NULL,
    is_master boolean DEFAULT false NOT NULL,
    is_ca boolean DEFAULT false NOT NULL,
    EXCLUDE (is_ca WITH =) WHERE (is_ca),
    EXCLUDE (is_master WITH =) WHERE (is_master)
);