CREATE TABLE resourcemanagers_dispatcher_dispatches (
    dispatch_id text PRIMARY KEY,
    resource_id text NOT NULL REFERENCES allocation_resources(resource_id) ON DELETE CASCADE NOT NULL,
    allocation_id text NOT NULL
);
