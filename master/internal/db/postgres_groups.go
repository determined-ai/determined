package db

func (db *PgDB) AddGroup() {
	return
}

func (db *PgDB) GroupByID(gid int) {
	return
}

func (db *PgDB) SearchGroups(userBelongsTo, groupIsOrDescendsFrom int) {
	return
}

func (db *PgDB) DeleteGroup(gid int) {
	return
}

func (db *PgDB) DeleteGroupRecursive(gid int) {
	return
}

func (db *PgDB) UpdateGroup() {
	return
}

func (db *PgDB) AddUserToGroup(gid, uid int) {
	return
}

func (db *PgDB) DeleteUserFromGroup(gid, uid int) {
	return
}

func (db *PgDB) AddUsersToGroup(gid int, uids ...int) {
	return
}

func (db *PgDB) DeleteUsersFromGroup(gid int, uids ...int) {
	return
}

func (db *PgDB) GetUsersInGroupRecursive(gid int) {
	return
}
