ALTER TABLE workspaces
	ADD CONSTRAINT namemin CHECK (char_length(name) >= 1);
ALTER TABLE workspaces
	ADD CONSTRAINT namemax CHECK (char_length(name) <= 80);
ALTER TABLE projects
	ADD CONSTRAINT namemin CHECK (char_length(name) >= 1);
ALTER TABLE projects
	ADD CONSTRAINT namemax CHECK (char_length(name) <= 80);
