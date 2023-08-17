package usergroup

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// addGroup adds a group to the database. Returns ErrDuplicateRow if a
// group already exists with the same name or ID. Will use db.Bun() if
// passed nil for idb.
func addGroup(ctx context.Context, idb bun.IDB, group Group) (Group, error) {
	if idb == nil {
		idb = db.Bun()
	}

	_, err := idb.NewInsert().Model(&group).Exec(ctx)
	return group, errors.Wrapf(db.MatchSentinelError(err), "Error creating group %s", group.Name)
}

// AddGroupWithMembers creates a group and adds members to it all in one transaction.
// If an empty user set is passed in, no transaction is used for performance reasons.
func AddGroupWithMembers(ctx context.Context, group Group, uids ...model.UserID) (Group,
	[]model.User, error,
) {
	if len(uids) == 0 {
		newGroup, err := addGroup(ctx, nil, group)
		return newGroup, nil, err
	}
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return Group{}, nil, errors.Wrapf(
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
		return Group{}, nil, err
	}

	idsToAdd := make([]model.UserID, 0, len(uids)+1)
	idsToAdd = append(idsToAdd, uids...)
	if group.OwnerID != 0 && !slices.Contains(idsToAdd, group.OwnerID) {
		idsToAdd = append(idsToAdd, group.OwnerID)
	}
	if len(idsToAdd) > 0 {
		err = AddUsersToGroupTx(ctx, tx, group.ID, idsToAdd...)
		if err != nil {
			return Group{}, nil, err
		}
	}

	users, err := UsersInGroupTx(ctx, tx, group.ID)
	if err != nil {
		return Group{}, nil, err
	}

	err = tx.Commit()
	if err != nil {
		return Group{}, nil, errors.Wrapf(
			db.MatchSentinelError(err),
			"Error committing changes to group %d",
			group.ID)
	}

	return group, users, nil
}

// GroupByIDTx looks for a group by id. Returns ErrNotFound if the group isn't found.
func GroupByIDTx(ctx context.Context, idb bun.IDB, gid int) (Group, error) {
	if idb == nil {
		idb = db.Bun()
	}
	var g Group
	err := idb.NewSelect().Model(&g).Where("id = ?", gid).Scan(ctx)
	if g.OwnerID != 0 {
		return Group{}, errors.Wrap(db.ErrNotFound, "cannot get a personal group")
	}

	return g, errors.Wrapf(db.MatchSentinelError(err), "Error getting group %d", gid)
}

// SearchGroups searches the database for groups. userBelongsTo is "optional"
// in that if a value < 1 is passed in, the parameter is ignored. SearchGroups
// does not return an error if no groups are found, as that is considered a
// successful search. SearchGroups includes personal groups which should not
// be exposed to an end user.
func SearchGroups(
	ctx context.Context, name string, userBelongsTo model.UserID, offset, limit int,
) (groups []Group, memberCounts []int32, tableRows int, err error) {
	query := SearchGroupsQuery(name, userBelongsTo, true)
	return SearchGroupsPaginated(ctx, query, offset, limit)
}

// SearchGroupsWithoutPersonalGroups searches the database for groups.
// userBelongsTo is "optional" in that if a value < 1 is passed in, the
// parameter is ignored. SearchGroups does not return an error if no groups
// are found, as that is considered a successful search.
func SearchGroupsWithoutPersonalGroups(
	ctx context.Context, name string, userBelongsTo model.UserID, offset, limit int,
) (groups []Group, memberCounts []int32, tableRows int, err error) {
	query := SearchGroupsQuery(name, userBelongsTo, false)
	return SearchGroupsPaginated(ctx, query, offset, limit)
}

// SearchGroupsQuery builds a query and returns it to the caller. userBelongsTo
// is "optional in that if a value < 1 is passed in, the parameter is ignored.
func SearchGroupsQuery(name string, userBelongsTo model.UserID,
	includePersonal bool,
) *bun.SelectQuery {
	var groups []Group
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
) (groups []Group, memberCounts []int32, tableRows int, err error) {
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
		Model(&Group{ID: gid}).
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
func UpdateGroupTx(ctx context.Context, idb bun.IDB, group Group) error {
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

// AddUsersToGroupTx adds users to a group by creating GroupMembership rows.
// Returns ErrNotFound if the group isn't found or ErrDuplicateRow if one
// of the users is already in the group. Will use db.Bun() if passed nil
// for idb.
func AddUsersToGroupTx(ctx context.Context, idb bun.IDB, gid int, uids ...model.UserID) error {
	if idb == nil {
		idb = db.Bun()
	}
	if _, err := GroupByIDTx(ctx, idb, gid); err != nil {
		return err
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
	if foundErr := db.MustHaveAffectedRows(res, err); foundErr != nil {
		sError := db.MatchSentinelError(foundErr)
		if errors.Is(sError, db.ErrNotFound) {
			return errors.Wrapf(sError,
				"Error adding %d user(s) to group %d because"+
					" one or more of them were not found", len(uids), gid)
		}
		return errors.Wrapf(sError, "Error when adding %d user(s) to group %d",
			len(uids), gid)
	}

	return nil
}

// RemoveUsersFromGroupTx removes users from a group. Removes nothing and
// returns ErrNotFound if the group or one of the users' membership rows
// aren't found.
func RemoveUsersFromGroupTx(ctx context.Context, idb bun.IDB, gid int, uids ...model.UserID) error {
	if _, err := GroupByIDTx(ctx, idb, gid); err != nil {
		return err
	}

	if len(uids) < 1 {
		return nil
	}

	if idb == nil {
		idb = db.Bun()
	}

	res, err := idb.NewDelete().Table("user_group_membership").
		Where("group_id = ?", gid).
		Where("user_id IN (?)", bun.In(uids)).
		Exec(ctx)
	if foundErr := db.MustHaveAffectedRows(res, err); foundErr != nil {
		sError := db.MatchSentinelError(foundErr)
		if errors.Is(sError, db.ErrNotFound) {
			return errors.Wrapf(sError,
				"Error removing %d user(s) from group %d because"+
					" one or more of them were not found", len(uids), gid)
		}
		return errors.Wrapf(sError, "Error when removing %d user(s) from group %d",
			len(uids), gid)
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
	err = UpdateGroupTx(ctx, tx, Group{
		ID:      gid,
		Name:    newName,
		OwnerID: oldGroup.OwnerID,
	})
	if err != nil {
		return nil, "", err
	}

	if len(addUsers) > 0 {
		err = AddUsersToGroupTx(ctx, tx, gid, addUsers...)
		if err != nil {
			return nil, "", err
		}

		err = UpdateUsersTimestampTx(ctx, tx, addUsers)
		if err != nil {
			return nil, "", err
		}
	}

	if len(removeUsers) > 0 {
		err = RemoveUsersFromGroupTx(ctx, tx, gid, removeUsers...)
		if err != nil {
			return nil, "", err
		}

		err = UpdateUsersTimestampTx(ctx, tx, removeUsers)
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

// UpdateUsersTimestampTx updates the user modified_at field to the present time.
func UpdateUsersTimestampTx(ctx context.Context, idb bun.IDB,
	uids []model.UserID,
) error {
	_, err := idb.NewUpdate().Table("users").
		Set("modified_at = NOW()").
		Where("id IN (?)", bun.In(uids)).
		Exec(ctx)
	if err != nil {
		return errors.Wrapf(db.MatchSentinelError(err),
			"Error updating modified_at timestamp for users")
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
