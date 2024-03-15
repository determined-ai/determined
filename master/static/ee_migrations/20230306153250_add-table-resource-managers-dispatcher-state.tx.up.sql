CREATE TABLE resourcemanagers_dispatcher_rm_state (
    id int UNIQUE DEFAULT(0),
    CONSTRAINT id_test CHECK (id = 0),
    disabled_agents text[] NOT NULL DEFAULT '{}'
);
