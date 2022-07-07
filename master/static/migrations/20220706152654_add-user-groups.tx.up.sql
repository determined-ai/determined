CREATE TABLE groups (
    id serial PRIMARY KEY,
    group_name text unique NOT NULL,
    user_id integer REFERENCES users (id) NULL
);

CREATE TABLE user_group_membership (
    user_id integer REFERENCES users (id),
    group_id integer REFERENCES groups (id),

    PRIMARY KEY (user_id, group_id)
);
