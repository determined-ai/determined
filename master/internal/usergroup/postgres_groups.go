package usergroup

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// addGroup adds a group to the database. Returns ErrDuplicateRow if a
// group already exists with the same name or ID. Will use db.Bun() if
// passed nil for idb.
func addGroup(ctx context.Context, idb bun.IDB, group model.Group) (model.Group, error) {
	if idb == nil {
		idb = db.Bun()
	}

	_, err := idb.NewInsert().Model(&group).Exec(ctx)
	return group, errors.Wrapf(db.MatchSentinelError(err), "Error creating group %s", group.Name)
}

// AddGroupWithMembers creates a group and adds members to it all in one transaction.
// If an empty user set is passed in, no transaction is used for performance reasons.
func AddGroupWithMembers(ctx context.Context, group model.Group, uids ...model.UserID) (model.Group,
	[]model.User, error,
) {
	if len(uids) == 0 {
		newGroup, err := addGroup(ctx, nil, group)
		return newGroup, nil, err
	}
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return model.Group{}, nil, errors.Wrapf(
			db.MatchSentinelError(err),
			"Error starting transaction for group %d creation",
			group.ID)
	}
	defer func() {
		txErr := tx.Rollback()
		if txErr != nil && txErr != sql.ErrTxDone {
			logrus.WithError(txErr).Error("error rolling back transaction in AddGroupWithMembers")
		}
	}()

	group, err = addGroup(ctx, tx, group)
	if err != nil {
		return model.Group{}, nil, err
	}

	idsToAdd := make([]model.UserID, 0, len(uids)+1)
	idsToAdd = append(idsToAdd, uids...)
	if group.OwnerID != 0 && !slices.Contains(idsToAdd, group.OwnerID) {
		idsToAdd = append(idsToAdd, group.OwnerID)
	}
	if len(idsToAdd) > 0 {
		err = AddUsersToGroupsTx(ctx, tx, []int{group.ID}, false, idsToAdd...)
		if err != nil {
			return model.Group{}, nil, err
		}
	}

	users, err := UsersInGroupTx(ctx, tx, group.ID)
	if err != nil {
		return model.Group{}, nil, err
	}

	err = tx.Commit()
	if err != nil {
		return model.Group{}, nil, errors.Wrapf(
			db.MatchSentinelError(err),
			"Error committing changes to group %d",
			group.ID)
	}

	return group, users, nil
}

// GroupByIDTx looks for a group by id. Returns ErrNotFound if the group isn't found.
func GroupByIDTx(ctx context.Context, idb bun.IDB, gid int) (model.Group, error) {
	if idb == nil {
		idb = db.Bun()
	}
	var g model.Group
	err := idb.NewSelect().Model(&g).Where("id = ?", gid).Scan(ctx)
	if g.OwnerID != 0 {
		return model.Group{}, errors.Wrap(db.ErrNotFound, "cannot get a personal group")
	}

	return g, errors.Wrapf(db.MatchSentinelError(err), "Error getting group %d", gid)
}

// ModifiableGroupsTx verifies that groups are in the DB and non-personal. Returns error if any group isn't found.
// Based on singular GroupByIDTx.
func ModifiableGroupsTx(ctx context.Context, idb bun.IDB, groups []int) error {
	if len(groups) == 0 {
		return nil
	}
	if idb == nil {
		idb = db.Bun()
	}
	count, err := idb.NewSelect().
		Table("groups").
		Where("user_id IS NULL").
		Where("id IN (?)", bun.In(groups)).
		Count(ctx)
	if len(groups) != count {
		return errors.Wrap(db.ErrNotFound, "group does not exist or is a personal group")
	}

	return errors.Wrapf(db.MatchSentinelError(err), "Error getting non-personal groups")
}

// SearchGroups searches the database for groups. userBelongsTo is "optional"
// in that if a value < 1 is passed in, the parameter is ignored. SearchGroups
// does not return an error if no groups are found, as that is considered a
// successful search. SearchGroups includes personal groups which should not
// be exposed to an end user.
func SearchGroups(
	ctx context.Context, name string, userBelongsTo model.UserID, offset, limit int,
) (groups []model.Group, memberCounts []int32, tableRows int, err error) {
	query := SearchGroupsQuery(name, userBelongsTo, true)
	return SearchGroupsPaginated(ctx, query, offset, limit)
}

// SearchGroupsWithoutPersonalGroups searches the database for groups.
// userBelongsTo is "optional" in that if a value < 1 is passed in, the
// parameter is ignored. SearchGroups does not return an error if no groups
// are found, as that is considered a successful search.
func SearchGroupsWithoutPersonalGroups(
	ctx context.Context, name string, userBelongsTo model.UserID, offset, limit int,
) (groups []model.Group, memberCounts []int32, tableRows int, err error) {
	query := SearchGroupsQuery(name, userBelongsTo, false)
	return SearchGroupsPaginated(ctx, query, offset, limit)
}

// SearchGroupsQuery builds a query and returns it to the caller. userBelongsTo
// is "optional in that if a value < 1 is passed in, the parameter is ignored.
func SearchGroupsQuery(name string, userBelongsTo model.UserID,
	includePersonal bool,
) *bun.SelectQuery {
	var groups []model.Group
	query := db.Bun().NewSelect().Model(&groups)
	if !includePersonal {
		query = query.Where("groups.user_id IS NULL")
	}

	if len(name) > 0 {
		query = query.Where("group_name = ?", name)
	}

	if userBelongsTo != 0 {
		query = query.Where(
			`EXISTS(SELECT 1
			FROM user_group_membership AS m
			WHERE m.group_id=groups.id AND m.user_id = ?)`,
			userBelongsTo)
	}
	return query
}

// SearchGroupsPaginated adds pagination arguments to a group search query and
// executes it. SearchGroupsPaginated does not return an error if no groups
// are found (that is a successful search).
func SearchGroupsPaginated(ctx context.Context,
	query *bun.SelectQuery, offset, limit int,
) (groups []model.Group, memberCounts []int32, tableRows int, err error) {
	paginatedQuery := db.PaginateBun(query, "id", db.SortDirectionAsc, offset, limit)

	err = paginatedQuery.Scan(ctx, &groups)
	if err != nil {
		return nil, nil, 0, err
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, nil, 0, err
	}

	var counts []int32
	err = paginatedQuery.Model(&counts).
		ColumnExpr("COUNT(ugm.user_id) AS num_members").
		Join("LEFT JOIN user_group_membership AS ugm ON groups.id=ugm.group_id").
		Group("id").
		Scan(ctx)
	if err != nil {
		return nil, nil, 0, err
	}

	searchResults := make([]*groupv1.GroupSearchResult, len(groups))
	for i, g := range groups {
		searchResults[i] = &groupv1.GroupSearchResult{
			Group:      g.Proto(),
			NumMembers: counts[i],
		}
	}

	return groups, counts, count, err
}

// DeleteGroup deletes a group from the database. Returns ErrNotFound if the
// group doesn't exist.
func DeleteGroup(ctx context.Context, gid int) error {
	res, err := db.Bun().NewDelete().
		Model(&model.Group{ID: gid}).
		WherePK().
		Where("user_id IS NULL"). // Cannot delete personal group.
		Exec(ctx)
	if foundErr := db.MustHaveAffectedRows(res, err); foundErr != nil {
		return errors.Wrapf(db.MatchSentinelError(foundErr), "Error deleting group %d", gid)
	}

	return nil
}

// UpdateGroupTx updates a group in the database. Returns ErrNotFound if the
// group isn't found.
func UpdateGroupTx(ctx context.Context, idb bun.IDB, group model.Group) error {
	if idb == nil {
		idb = db.Bun()
	}
	res, err := idb.NewUpdate().
		Model(&group).
		WherePK().
		Where("user_id IS NULL"). // Cannot update personal group.
		Exec(ctx)

	return errors.Wrapf(
		db.MatchSentinelError(db.MustHaveAffectedRows(res, err)),
		"Error updating group %d name",
		group.ID)
}

// AddUsersToGroupsTx adds users to groups by creating GroupMembership rows.
// Returns ErrNotFound if the group isn't found or ErrDuplicateRow if one
// of the users is already in the group (unless ignoreDuplicates).
// Will use db.Bun() if passed nil for idb.
func AddUsersToGroupsTx(ctx context.Context, idb bun.IDB, groups []int, ignoreDuplicates bool,
	uids ...model.UserID,
) error {
	if idb == nil {
		idb = db.Bun()
	}

	if err := ModifiableGroupsTx(ctx, idb, groups); err != nil {
		return err
	}

	if len(uids) < 1 {
		return nil
	}

	groupMem := make([]model.GroupMembership, 0, len(uids)*len(groups))
	for _, uid := range uids {
		for _, gid := range groups {
			groupMem = append(groupMem, model.GroupMembership{
				UserID:  uid,
				GroupID: gid,
			})
		}
	}

	query := idb.NewInsert().Model(&groupMem)
	if ignoreDuplicates {
		query = query.On("CONFLICT(user_id, group_id) DO NOTHING")
		_, err := query.Exec(ctx)
		if err != nil {
			return errors.Wrapf(err,
				"Error adding %d user(s) to %d group(s)", len(uids), len(groups))
		}
	} else {
		res, err := query.Exec(ctx)
		if foundErr := db.MustHaveAffectedRows(res, err); foundErr != nil {
			sError := db.MatchSentinelError(foundErr)
			if errors.Is(sError, db.ErrNotFound) {
				return errors.Wrapf(sError,
					"Error adding %d user(s) to %d group(s) because"+
						" one or more of them were not found", len(uids), len(groups))
			}
			return errors.Wrapf(sError, "Error when adding %d user(s) to %d group(s)",
				len(uids), len(groups))
		}
	}

	err := UpdateUsersTimestampTx(ctx, idb, uids)
	if err != nil {
		return fmt.Errorf("error when updating users timestamps: %w", err)
	}

	return nil
}

// RemoveUsersFromGroupsTx removes users from a group. Removes nothing and
// returns ErrNotFound if the group or all of the membership rows
// aren't found.
func RemoveUsersFromGroupsTx(ctx context.Context, idb bun.IDB, groups []int,
	uids ...model.UserID,
) error {
	if idb == nil {
		idb = db.Bun()
	}

	if err := ModifiableGroupsTx(ctx, idb, groups); err != nil {
		return err
	}

	if len(uids) < 1 {
		return nil
	}

	var changeRecords []int32
	_, err := idb.NewDelete().Model(&changeRecords).
		Table("user_group_membership").
		Where("group_id IN (?)", bun.In(groups)).
		Where("user_id IN (?)", bun.In(uids)).
		Returning("user_id").
		Exec(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error when removing %d user(s) from %d group(s)",
			len(uids), len(groups))
	}

	if len(changeRecords) == 0 {
		return errors.Wrapf(db.ErrNotFound,
			"Error removing %d user(s) from %d group(s) because"+
				" none were members of these groups", len(uids), len(groups))
	}

	err = UpdateUsersTimestampTx(ctx, idb, uids)
	if err != nil {
		return fmt.Errorf("error when updating users timestamps: %w", err)
	}

	return nil
}

// UpdateGroupAndMembers updates a group and adds or removes members all in one transaction.
func UpdateGroupAndMembers(
	ctx context.Context,
	gid int, name string,
	addUsers,
	removeUsers []model.UserID,
) ([]model.User, string, error) {
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, "", errors.Wrapf(
			db.MatchSentinelError(err),
			"Error starting transaction for group %d update",
			gid)
	}
	defer func() {
		txErr := tx.Rollback()
		if txErr != nil && txErr != sql.ErrTxDone {
			logrus.WithError(txErr).Error("error rolling back transaction in UpdateGroupAndMembers")
		}
	}()

	oldGroup, err := GroupByIDTx(ctx, tx, gid)
	if err != nil {
		return nil, "", err
	}

	newName := oldGroup.Name
	if name != "" {
		newName = name
	}
	err = UpdateGroupTx(ctx, tx, model.Group{
		ID:      gid,
		Name:    newName,
		OwnerID: oldGroup.OwnerID,
	})
	if err != nil {
		return nil, "", err
	}

	if len(addUsers) > 0 {
		err = AddUsersToGroupsTx(ctx, tx, []int{gid}, false, addUsers...)
		if err != nil {
			return nil, "", err
		}
	}

	if len(removeUsers) > 0 {
		err = RemoveUsersFromGroupsTx(ctx, tx, []int{gid}, removeUsers...)
		if err != nil {
			return nil, "", err
		}
	}

	users, err := UsersInGroupTx(ctx, tx, gid)
	if err != nil {
		return nil, "", err
	}

	err = tx.Commit()
	if err != nil {
		return nil, "", errors.Wrapf(db.MatchSentinelError(err),
			"Error committing changes to group %d", gid)
	}

	return users, newName, nil
}

// UpdateGroupsForMultipleUsers adds and removes group associations for multiple members.
func UpdateGroupsForMultipleUsers(
	ctx context.Context,
	modUsers []model.UserID,
	addGroups []int,
	removeGroups []int,
) error {
	return db.Bun().RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			if len(addGroups) > 0 {
				err = AddUsersToGroupsTx(ctx, tx, addGroups, true, modUsers...)
				if err != nil {
					return err
				}
			}

			if len(removeGroups) > 0 {
				err = RemoveUsersFromGroupsTx(ctx, tx, removeGroups, modUsers...)
				if err != nil {
					return err
				}
			}

			return ni
		})
}

// UpdateUsersTimestampTx updates the user modified_at field to the present time.
func UpdateUsersTimestampTx(ctx context.Context, idb bun.IDB,
	uids []model.UserID,
) error {
	_, err := idb.NewUpdate().Table("users").
		Set("modified_at = NOW()").
		Where("id IN (?)", bun.In(uids)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("updating modified_at timestamp for users: %w",
			db.MatchSentinelError(err))
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

	return users, errors.Wrapf(db.MatchSentinelError(err), "Error getting group %d info", gid)
}
