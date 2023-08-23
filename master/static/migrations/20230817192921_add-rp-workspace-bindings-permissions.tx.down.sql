DELETE from permission_assignments WHERE permission_id IN (4007, 10001);

DELETE from permissions WHERE name IN (
    'set default resource pool on workspace',
    'modify rp workspace bindings'
);
