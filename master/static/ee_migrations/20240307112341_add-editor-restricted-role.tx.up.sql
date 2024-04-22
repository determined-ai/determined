INSERT INTO roles(id, role_name) VALUES  (7, 'EditorRestricted');

INSERT INTO permission_assignments(permission_id, role_id)
    SELECT permission_id, 7 FROM permission_assignments 
    WHERE 
        role_id = (SELECT id FROM roles WHERE role_name = 'Editor') 
        AND permission_id NOT IN (SELECT id FROM permissions 
            WHERE name = 'create notebooks/shells/commands' 
                OR name = 'update notebooks/shells/commands');
