DELETE FROM permissions_assignments WHERE permission_id IN (
    8001, 8002
);

DELETE FROM permissions WHERE id IN (
    8001, 8002
);
