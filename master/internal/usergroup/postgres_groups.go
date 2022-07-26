package usergroup

import (
	"context"
	"database/sql"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// addGroup adds a group to the database. Returns ErrDuplicateRow if a
// group already exists with the same name or ID. Will use db.Bun() if
// passed nil for idb.
func addGroup(ctx context.Context, idb bun.IDB, group Group) (Group, error) {
	if idb == nil {
		idb = db.Bun()
	}

	_, err := idb.NewInsert().Model(&group).Exec(ctx)
	return group, matchSentinelError(err)
}

// AddGroupWithMembers creates a group and adds members to it all in one transaction.
// If an empty user set is passed in, no transaction is used for performance reasons.
func AddGroupWithMembers(ctx context.Context, group Group, uids ...model.UserID) (Group, []model.User, error) {
	if len(uids) == 0 {
		group, err := addGroup(ctx, nil, group)
		return group, nil, err
	}

	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return Group{}, nil, err
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			logrus.WithError(err).Error("error rolling back transaction in AddGroupWithMembers")
		}
	}()

	group, err = addGroup(ctx, tx, group)
	if err != nil {
		return Group{}, nil, err
	}

	err = AddUsersToGroupTx(ctx, tx, group.ID, uids...)
	if err != nil {
		return Group{}, nil, err
	}

	users, err := UsersInGroupTx(ctx, tx, group.ID)
	if err != nil {
		return Group{}, nil, err
	}

	err = tx.Commit()
	if err != nil {
		return Group{}, nil, err
	}

	return group, users, nil
}

// GroupByID looks for a group by id. Returns ErrNotFound if the group isn't found.
func GroupByID(ctx context.Context, gid int) (Group, error) {
	var g Group
	err := db.Bun().NewSelect().Model(&g).Where("id = ?", gid).Scan(ctx)

	return g, matchSentinelError(err)
}

// SearchGroups searches the database for groups. userBelongsTo is "optional"
// in that if a value < 1 is passed in, the parameter is ignored. SearchGroups
// does not return an error if no groups are found, as that is considered a
// successful search.
func SearchGroups(ctx context.Context,
	name string,
	userBelongsTo model.UserID,
	offset, limit int,
) ([]Group, int, error) {
	var groups []Group
	query := db.Bun().NewSelect().Model(&groups)

	if len(name) > 0 {
		query = query.Where("group_name = ?", name)
	}

	if userBelongsTo > 0 {
		query = query.
			Join("INNER JOIN user_group_membership AS ugm ON ugm.group_id=groups.id").
			Where("ugm.user_id = ?", userBelongsTo)
	}

	err := db.PaginateBun(query, "", "", offset, limit).Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return groups, count, err
}

// DeleteGroup deletes a group from the database. Returns ErrNotFound if the
// group doesn't exist.
func DeleteGroup(ctx context.Context, gid int) error {
	res, err := db.Bun().NewDelete().Model(&Group{ID: gid}).WherePK().Exec(ctx)
	if foundErr := mustHaveAffectedRows(res, err); foundErr != nil {
		return matchSentinelError(foundErr)
	}

	return nil
}

// UpdateGroup updates a group in the database. Returns ErrNotFound if the
// group isn't found.
func UpdateGroup(ctx context.Context, group Group) error {
	res, err := db.Bun().NewUpdate().Model(&group).WherePK().Exec(ctx)

	return matchSentinelError(mustHaveAffectedRows(res, err))
}

// AddUsersToGroupTx adds users to a group by creating GroupMembership rows.
// Returns ErrNotFound if the group isn't found or ErrDuplicateRow if one
// of the users is already in the group. Will use db.Bun() if passed nil
// for idb.
func AddUsersToGroupTx(ctx context.Context, idb bun.IDB, gid int, uids ...model.UserID) error {
	if idb == nil {
		idb = db.Bun()
	}

	if len(uids) < 1 {
		return nil
	}

	groupMem := make([]GroupMembership, 0, len(uids))
	for _, uid := range uids {
		groupMem = append(groupMem, GroupMembership{
			UserID:  uid,
			GroupID: gid,
		})
	}

	res, err := idb.NewInsert().Model(&groupMem).Exec(ctx)
	if foundErr := mustHaveAffectedRows(res, err); foundErr != nil {
		return matchSentinelError(foundErr)
	}

	return nil
}

// RemoveUsersFromGroup removes users from a group. Removes nothing and
// returns ErrNotFound if the group or one of the users' membership rows
// aren't found.
func RemoveUsersFromGroup(ctx context.Context, gid int, uids ...model.UserID) error {
	if len(uids) < 1 {
		return nil
	}

	res, err := db.Bun().NewDelete().Table("user_group_membership").
		Where("group_id = ?", gid).
		Where("user_id IN (?)", bun.In(uids)).
		Exec(ctx)
	if foundErr := mustHaveAffectedRows(res, err); foundErr != nil {
		return matchSentinelError(foundErr)
	}

	return nil
}

// UsersInGroupTx searches for users that belong to a group and returns them.
// Does not return ErrNotFound if none are found, as that is considered a
// successful search. Will use db.Bun() if passed nil for idb.
func UsersInGroupTx(ctx context.Context, idb bun.IDB, gid int) ([]model.User, error) {
	if idb == nil {
		idb = db.Bun()
	}

	var users []model.User
	err := idb.NewSelect().Model(&users).
		Join(`INNER JOIN user_group_membership AS ugm ON "user"."id"=ugm.user_id`).
		Where("ugm.group_id = ?", gid).
		Scan(ctx)

	return users, err
}

func matchSentinelError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNotFound
	}

	switch pgErrCode(err) {
	case db.CodeForeignKeyViolation:
		return db.ErrNotFound
	case db.CodeUniqueViolation:
		return db.ErrDuplicateRecord
	}

	return err
}

// mustHaveAffectedRows checks if bun has affected rows in a table or not.
// Returns ErrNotFound if no rows were affected and returns the provided error otherwise.
func mustHaveAffectedRows(result sql.Result, err error) error {
	if err == nil {
		rowsAffected, affectedErr := result.RowsAffected()
		if affectedErr != nil {
			return affectedErr
		}
		if rowsAffected == 0 {
			return db.ErrNotFound
		}
	}

	return err
}

func pgErrCode(err error) string {
	if e, ok := err.(*pgconn.PgError); ok {
		return e.Code
	}

	return ""
}
