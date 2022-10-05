ALTER TABLE workspaces
	ADD COLUMN uid integer,
	ADD COLUMN user_ text,
	ADD COLUMN gid integer,
	ADD COLUMN group_ text,
	ADD CONSTRAINT uidnull CHECK ((uid IS NULL) = (user_ IS NULL)),
	ADD CONSTRAINT gidnull CHECK ((gid IS NULL) = (group_ IS NULL));
