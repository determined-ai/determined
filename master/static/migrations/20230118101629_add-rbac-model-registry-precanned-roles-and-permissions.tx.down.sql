DELETE from permissions_assignments WHERE permission_id IN (
    7001, 
    7002, 
    7003
    ); 

DELETE from permissions WHERE name IN (
    'view model registry', 
    'edit model registry', 
    'create model registry'
    );