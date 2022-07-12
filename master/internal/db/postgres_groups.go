package db

import (
	"context"
	"database/sql"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AddGroup adds a group to the database. Returns ErrDuplicateRow if a
// group already exists with the same name or ID.
func (db *PgDB) AddGroup(ctx context.Context, group model.Group) (model.Group, error) {
	_, err := Bun().NewInsert().Model(&group).Exec(ctx)
	return group, matchSentinelError(err)
}

// GroupByID looks for a group by id. Returns ErrNotFound if the group isn't found.
func (db *PgDB) GroupByID(ctx context.Context, gid int) (model.Group, error) {
	var g model.Group
	err := Bun().NewSelect().Model(&g).Where("id = ?", gid).Scan(ctx)

	return g, matchSentinelError(err)
}

// SearchGroups searches the database for groups. userBelongsTo is "optional"
// in that if a value < 1 is passed in, the parameter is ignored. SearchGroups
// does not return an error if no groups are found, as that is considered a
// successful search.
func (db *PgDB) SearchGroups(ctx context.Context,
	userBelongsTo model.UserID) ([]model.Group, error) {
	var groups []model.Group
	query := Bun().NewSelect().Model(&groups)

	if userBelongsTo > 0 {
		query = query.
			Join("INNER JOIN user_group_membership AS ugm ON ugm.group_id=groups.id").
			Where("ugm.user_id = ?", userBelongsTo)
	}

	err := query.Scan(ctx)

	return groups, err
}

// DeleteGroup deletes a group from the database. Returns ErrNotFound if the
// group doesn't exist.
func (db *PgDB) DeleteGroup(ctx context.Context, gid int) error {
	tx, err := Bun().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err := tx.Rollback()
		if err != sql.ErrTxDone && err != nil {
			log.WithError(err).
				WithField("groupID", gid).
				Error("error rolling back transaction in DeleteGroup")
		}
	}()

	_, err = tx.NewDelete().
		Table("user_group_membership").
		Where("group_id = ?", gid).
		Exec(ctx)
	if err != nil {
		return err
	}

	res, err := tx.NewDelete().Model(&model.Group{ID: gid}).WherePK().Exec(ctx)
	if foundErr := mustHaveAffectedRows(res, err); foundErr != nil {
		return matchSentinelError(foundErr)
	}

	err = tx.Commit()
	if err != nil {
		return matchSentinelError(err)
	}

	return nil
}

// UpdateGroup updates a group in the database. Returns ErrNotFound if the
// group isn't found.
func (db *PgDB) UpdateGroup(ctx context.Context, group model.Group) error {
	res, err := Bun().NewUpdate().Model(&group).WherePK().Exec(ctx)

	return matchSentinelError(mustHaveAffectedRows(res, err))
}

// AddUsersToGroup adds users to a group by creating GroupMembership rows.
// Returns ErrNotFound if the group isn't found or ErrDuplicateRow if one
// of the users is already in the group.
func (db *PgDB) AddUsersToGroup(ctx context.Context, gid int, uids ...model.UserID) error {
	if len(uids) < 1 {
		return nil
	}

	groupMem := make([]model.GroupMembership, 0, len(uids))
	for _, uid := range uids {
		groupMem = append(groupMem, model.GroupMembership{
			UserID:  uid,
			GroupID: gid,
		})
	}

	tx, err := Bun().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err := tx.Rollback()
		if err != sql.ErrTxDone && err != nil {
			log.WithError(err).
				WithField("groupID", gid).
				WithField("userIDs", uids).
				Error("error rolling back transaction in AddUsersToGroup")
		}
	}()

	res, err := tx.NewInsert().Model(&groupMem).Exec(ctx)
	if foundErr := mustHaveAffectedRows(res, err); foundErr != nil {
		return matchSentinelError(foundErr)
	}

	err = tx.Commit()
	return matchSentinelError(err)
}

// RemoveUsersFromGroup removes users from a group. Removes nothing and
// returns ErrNotFound if the group or one of the users' membership rows
// aren't found.
func (db *PgDB) RemoveUsersFromGroup(ctx context.Context, gid int, uids ...model.UserID) error {
	if len(uids) < 1 {
		return nil
	}

	tx, err := Bun().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err := tx.Rollback()
		if err != sql.ErrTxDone && err != nil {
			log.WithError(err).
				WithField("groupID", gid).
				WithField("userIDs", uids).
				Error("error rolling back transaction in RemoveUsersFromGroup")
		}
	}()

	res, err := tx.NewDelete().Table("user_group_membership").
		Where("group_id = ?", gid).
		Where("user_id IN (?)", bun.In(uids)).
		Exec(ctx)
	if foundErr := mustHaveAffectedRows(res, err); foundErr != nil {
		return matchSentinelError(foundErr)
	}

	err = tx.Commit()
	return matchSentinelError(err)
}

// GetUsersInGroup searches for users that belong to a group and returns them.
// Does not return ErrNotFound if none are found, as that is considered a
// successful search.
func (db *PgDB) GetUsersInGroup(ctx context.Context, gid int) ([]model.User, error) {
	var users []model.User
	err := Bun().NewSelect().Table("users").Model(&users).
		Join("INNER JOIN user_group_membership AS ugm ON users.id=ugm.user_id").
		Where("ugm.group_id = ?", gid).
		Scan(ctx)

	return users, err
}

func matchSentinelError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	switch pgErrCode(err) {
	case foreignKeyViolation:
		return ErrNotFound
	case uniqueViolation:
		return ErrDuplicateRecord
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
			return ErrNotFound
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
