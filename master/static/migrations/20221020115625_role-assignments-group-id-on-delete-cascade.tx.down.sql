ALTER TABLE role_assignments
  DROP CONSTRAINT role_assignments_group_id_fkey,
  ADD CONSTRAINT role_assignments_group_id_fkey
	FOREIGN KEY (group_id) REFERENCES groups (id);
