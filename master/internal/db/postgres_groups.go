package db

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/jackc/pgconn"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TODO: return db.ErrDuplicateRecord when record exists
func (db *PgDB) AddGroup(ctx context.Context, group model.Group) (model.Group, error) {
	_, err := Bun().NewInsert().Model(&group).Exec(ctx)
	return group, err
}

func (db *PgDB) GroupByID(ctx context.Context, gid int) (model.Group, error) {
	var g model.Group
	err := Bun().NewSelect().Model(&g).Where("id = ?", gid).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return g, ErrNotFound
	}
	return g, err
}

func (db *PgDB) SearchGroups(ctx context.Context, userBelongsTo model.UserID) ([]model.Group, error) {
	var groups []model.Group
	query := Bun().NewSelect().Model(&groups).Distinct()

	if userBelongsTo > 0 {
		query = query.
			Join("INNER JOIN user_group_membership AS ugm ON ugm.group_id=groups.id").
			Where("ugm.user_id = ?", userBelongsTo)
	}

	err := query.Scan(ctx)

	return groups, err
}

func (db *PgDB) DeleteGroup(ctx context.Context, gid int) error {
	res, err := Bun().NewDelete().Table("user_group_membership").Where("group_id = ?", gid).Exec(ctx)

	if foundErr := checkIfFound(res, err); foundErr != nil {
		return foundErr
	}

	res, err = Bun().NewDelete().Table("groups").Where("id = ?", gid).Exec(ctx)
	return checkIfFound(res, err)
}

func (db *PgDB) UpdateGroup(ctx context.Context, group model.Group) error {
	res, err := Bun().NewUpdate().Model(&group).WherePK().Exec(ctx)

	return checkIfFound(res, err)
}

// TODO: return db.ErrDuplicateRecord when record exists
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

	tx, err := Bun().BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.NewInsert().Model(&groupMem).Exec(ctx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	return checkIfFound(res, err)
}

func (db *PgDB) RemoveUsersFromGroup(ctx context.Context, gid int, uids ...model.UserID) error {
	if len(uids) < 1 {
		return nil
	}

	tx, err := Bun().BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.NewDelete().Table("user_group_membership").
		Where("group_id = ?", gid).
		Where("user_id IN (?)", bun.In(uids)).
		Exec(ctx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	return checkIfFound(res, err)
}

func (db *PgDB) GetUsersInGroup(ctx context.Context, gid int) ([]model.User, error) {
	var users []model.User
	err := Bun().NewSelect().Table("users").Model(&users).
		Join("INNER JOIN user_group_membership AS ugm ON users.id=ugm.user_id").
		Where("ugm.group_id = ?", gid).
		Scan(ctx)

	return users, err
}

// TODO: actually finish this and test it (integration?)
// expected: *errors.fundamental(not found)
// actual  : *pgconn.PgError(&pgconn.PgError{Severity:"ERROR", Code:"23503", Message:"insert or update on table \"user_group_membership\" violates foreign key constraint \"user_group_membership_user_id_fkey\"", Detail:"Key (user_id)=(125674576) is not present in table \"users\".", Hint:"", Position:0, InternalPosition:0, InternalQuery:"", Where:"", SchemaName:"public", TableName:"user_group_membership", ColumnName:"", DataTypeName:"", ConstraintName:"user_group_membership_user_id_fkey", File:"ri_triggers.c", Line:3266, Routine:"ri_ReportViolation"})
func isNotFoundErr(err error) bool {
	if e, ok := err.(*pgconn.PgError); ok {
		if e.Code == "23503" {
			return true
		}
	}
	return false
}

// checkIfFound checks if bun has affected rows in a table or not.
// Returns ErrNotFound if no rows were affected and returns the provided error otherwise
func checkIfFound(result sql.Result, err error) error {
	if err == nil {
		rowsAffected, affectedErr := result.RowsAffected()
		if affectedErr == nil && rowsAffected == 0 {
			return ErrNotFound
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
