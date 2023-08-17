//go:build integration
// +build integration

package usergroup

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestUserGroups(t *testing.T) {
	ctx := context.Background()
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, pathToMigrations)

	t.Cleanup(func() { cleanUp(ctx, t) })
	setUp(ctx, t, pgDB)

	t.Run("group creation", func(t *testing.T) {
		_, _, err := AddGroupWithMembers(ctx, testGroup)
		require.NoError(t, err, "failed to create group")
	})

	t.Run("search groups", func(t *testing.T) {
		groups, _, count, err := SearchGroups(ctx, "", 0, 0, 0)
		require.NoError(t, err, "failed to search for groups")
		require.GreaterOrEqual(t, count, len(testGroups), "search returned the wrong count")

		index := groupsContain(groups, testGroup.ID)
		require.NotEqual(t, -1, index, "Expected groups to contain the new one")
		foundGroup := groups[index]
		require.Equal(t, testGroup.Name, foundGroup.Name,
			"Expected found group to have the same name as the one we created")

		groups, _, count, err = SearchGroups(ctx, testGroup.Name, 0, 0, 0)
		require.NoError(t, err, "failed to search for groups")
		require.Equal(t, 1, count, "search returned the wrong count")
		require.NotEmpty(t, groups, "failed to find group by name")
		require.Len(t, groups, 1, "failed to narrow search to just matching name")
		require.Equal(t, testGroup.Name, groups[0].Name, "failed to find the correct group")
	})

	t.Run("find group by id", func(t *testing.T) {
		foundGroup, err := GroupByIDTx(ctx, nil, testGroup.ID)
		require.NoError(t, err, "failed to find group by id")
		require.Equal(t, testGroup.Name, foundGroup.Name,
			"Expected found group to have the same name as the one we created")
	})

	t.Run("update group", func(t *testing.T) {
		// Put it back the way it was when we're done
		defer func(name string) {
			testGroup.Name = name
			err := UpdateGroupTx(ctx, nil, testGroup)
			require.NoError(t, err,
				"failed to put things back how they were after testing UpdateGroup")
		}(testGroup.Name)

		newName := "kljhadsflkgjhjklsfhgasdhj"
		testGroup.Name = newName
		err := UpdateGroupTx(ctx, nil, testGroup)
		require.NoError(t, err, "failed to update group")

		foundGroup, err := GroupByIDTx(ctx, nil, testGroup.ID)
		require.NoError(t, err, "failed to find group by id")
		require.Equal(t, newName, foundGroup.Name, "Expected found group to have the new name")
	})

	t.Run("add users to group", func(t *testing.T) {
		err := AddUsersToGroupTx(ctx, nil, testGroup.ID, testUser.ID)
		require.NoError(t, err, "failed to add users to group")

		users, err := UsersInGroupTx(ctx, nil, testGroup.ID)
		require.NoError(t, err, "failed to search for users that belong to group")
		require.Len(t, users, 1, "failed to return only the set of users in the group")

		index := usersContain(users, testUser.ID)
		require.NotEqual(t, -1, index, "Expected users in group to contain the newly added one")

		require.Equal(t, users[index].ModifiedAt, time.Now(), "Users.modified_at not updated when adding to group")
	})

	t.Run("search groups by user membership", func(t *testing.T) {
		groups, _, count, err := SearchGroupsWithoutPersonalGroups(ctx, "", testUser.ID, 0, 0)
		require.NoError(t, err, "failed to search for groups that user belongs to")

		index := groupsContain(groups, testGroup.ID)
		require.Equal(t, 2, count, "group search returned wrong count")
		require.NotEqual(t, -1, index,
			"Group user was added to not found when searching by user membership")
	})

	t.Run("manually edit groups query", func(t *testing.T) {
		query := SearchGroupsQuery("", testUser.ID, false)
		query = query.Where("group_name = ?", testGroup.Name)
		groups, _, count, err := SearchGroupsPaginated(ctx, query, 0, 0)
		require.NoError(t, err, "failed to search for group in modified query")
		require.Equal(t, 1, count, "modified group search returned wrong count")
		index := groupsContain(groups, testGroup.ID)
		require.NotEqual(t, -1, index,
			"Group user was added to not found when searching by user membership")
	})

	t.Run("remove users from group", func(t *testing.T) {
		err := RemoveUsersFromGroupTx(ctx, nil, testGroup.ID, -500)
		require.True(t, errors.Is(err, db.ErrNotFound),
			"failed to return ErrNotFound when removing non-existent users from group")

		err = RemoveUsersFromGroupTx(ctx, nil, testGroup.ID, testUser.ID, -500)
		require.NoError(t, err,
			"erroneously returned error when trying to remove a mix of users in a group and not")

		users, err := UsersInGroupTx(ctx, nil, testGroup.ID)
		require.NoError(t, err, "failed to look for users in group")

		i := usersContain(users, testUser.ID)
		require.Equal(t, -1, i, "User found in group after removing them from it")
		require.Equal(t, testUser.ModifiedAt, time.Now(), "Users.modified_at not updated when removed from group")

		err = RemoveUsersFromGroupTx(ctx, nil, testGroup.ID, testUser.ID)
		require.True(t, errors.Is(err, db.ErrNotFound),
			"failed to return ErrNotFound when trying to remove users from group they're not in")
	})

	t.Run("partial success on adding users to a group results in tx rollback and ErrNotFound",
		func(t *testing.T) {
			err := AddUsersToGroupTx(ctx, nil, testGroup.ID, testUser.ID, 125674576, 12934728, 0, -15)
			require.True(t, errors.Is(err, db.ErrNotFound),
				"didn't return ErrNotFound when adding non-existent users to a group")

			users, err := UsersInGroupTx(ctx, nil, testGroup.ID)
			require.NoError(t, err, "failed to search for users that belong to group")

			index := usersContain(users, testUser.ID)
			require.Equal(t, -1, index,
				"Expected users in group not to contain the one added in the erroring call")
		})

	t.Run("AddUsersToGroup fails with ErrNotFound when attempting "+
		"to add users to a non-existent group", func(t *testing.T) {
		err := AddUsersToGroupTx(ctx, nil, -500, testUser.ID)
		require.True(t, errors.Is(err, db.ErrNotFound),
			"didn't return ErrNotFound when trying to add users to a non-existent group")
	})

	t.Run("Deleting a group that doesn't exist results in ErrNotFound", func(t *testing.T) {
		err := DeleteGroup(ctx, -500)
		require.True(t, errors.Is(err, db.ErrNotFound),
			"didn't return ErrNotFound when trying to delete a non-existent group")
	})

	t.Run("Deleting a group that has users should work", func(t *testing.T) {
		tmpGroup := testGroup
		tmpGroup.ID++
		tmpGroup.Name += tmpGroup.Name

		_, _, err := AddGroupWithMembers(ctx, tmpGroup)
		require.NoError(t, err, "failed to create group")

		err = AddUsersToGroupTx(ctx, nil, tmpGroup.ID, testUser.ID)
		require.NoError(t, err, "failed to add user to group")

		err = DeleteGroup(ctx, tmpGroup.ID)
		require.NoError(t, err, "errored when deleting group")

		_, err = GroupByIDTx(ctx, nil, tmpGroup.ID)
		require.True(t, errors.Is(err, db.ErrNotFound),
			"deleted group should not be found, and ErrNotFound returned")
	})

	t.Run("AddGroup returns ErrDuplicateRecord when creating a group that already exists",
		func(t *testing.T) {
			_, _, err := AddGroupWithMembers(ctx, testGroupStatic)
			require.True(t, errors.Is(err, db.ErrDuplicateRecord), "didn't return ErrDuplicateRecord")
		})

	t.Run("AddUsersToGroup returns ErrDuplicateRecord when adding users to a "+
		"group they're already in", func(t *testing.T) {
		err := AddUsersToGroupTx(ctx, nil, testGroupStatic.ID, testUser.ID)
		require.True(t, errors.Is(err, db.ErrDuplicateRecord),
			"should have returned ErrDuplicateRecord")
	})

	t.Run("Static test group should exist at the end and test user should be in it",
		func(t *testing.T) {
			_, err := GroupByIDTx(ctx, nil, testGroupStatic.ID)
			require.NoError(t, err, "errored while getting static test group")

			users, err := UsersInGroupTx(ctx, nil, testGroupStatic.ID)
			require.NoError(t, err, "failed to search for users that belong to static group")

			index := usersContain(users, testUser.ID)
			require.NotEqual(t, -1, index, "Expected users in static group to contain the test user")
		})

	t.Run("search group with offsets and limits", func(t *testing.T) {
		answerGroups, _, count, err := SearchGroups(ctx, "", 0, 0, 3)
		require.NoError(t, err, "failed to search for groups")
		require.LessOrEqual(t, len(answerGroups), 3, "limit was not respected")
		require.GreaterOrEqual(t, count, len(testGroups), "returned wrong count of groups")

		groups, _, count, err := SearchGroups(ctx, "", 0, 0, 1)
		require.NoError(t, err, "failed to search for groups")
		require.GreaterOrEqual(t, count, len(testGroups), "returned wrong count of groups")
		require.Len(t, groups, 1, "limit was not respected")
		index := groupsContain(groups, answerGroups[0].ID)
		require.NotEqual(t, -1, index, "Expected groups to contain the new one")
		foundGroup := groups[index]
		require.Equal(t, answerGroups[0].Name, foundGroup.Name,
			"Expected found group to have the same name as the first answerGroup")
		require.Equal(t, 1, len(groups), "Expected no more than one group to have been returned")

		groups, _, count, err = SearchGroups(ctx, "", 0, 1, 2)
		require.NoError(t, err, "failed to search for groups")
		require.GreaterOrEqual(t, count, len(testGroups), "search returned the wrong count")
		require.LessOrEqual(t, len(groups), 2,
			"Expected no more than two groups to have been returned")
		index = groupsContain(groups, answerGroups[1].ID)
		require.NotEqual(t, -1, index, "Expected groups to contain the new one")
		foundGroup = groups[index]
		require.Equal(t, answerGroups[1].Name, foundGroup.Name,
			"Expected found group to have the same name as the second answerGroup")
	})

	t.Run("AddGroupWithMembers rolls back transactions", func(t *testing.T) {
		const tempGroupName = "tempGroupName"
		g, _, err := AddGroupWithMembers(ctx, Group{Name: tempGroupName}, 1, 2, 3, 1856109534)
		// Just in case this fails, clean up.
		defer func(g Group) {
			if g.ID != 0 {
				require.NoError(t, DeleteGroup(ctx, g.ID))
			}
		}(g)
		require.True(t, errors.Is(err, db.ErrNotFound), "error")

		groups, _, count, err := SearchGroups(ctx, tempGroupName, 0, 0, 0)
		require.NoError(t, err, "error searching for groups to verify rollback")
		require.Equal(t, 0, count, "should be zero matching groups in the DB")
		require.Equal(t, 0, len(groups), "should be zero matching groups returned")
	})

	t.Run("update groups and memberships", func(t *testing.T) {
		updateTestUser1 := model.User{
			ID:       9861724,
			Username: fmt.Sprintf("IntegrationTest#{9861724}"),
			Admin:    false,
			Active:   false,
		}
		updateTestUser2 := model.User{
			ID:       16780345,
			Username: fmt.Sprintf("IntegrationTest#{16780345}"),
			Admin:    false,
			Active:   false,
		}

		t.Cleanup(func() {
			_ = deleteUser(ctx, updateTestUser1.ID)
			_ = deleteUser(ctx, updateTestUser2.ID)
		})

		_, err := pgDB.AddUser(&updateTestUser1, nil)
		require.NoError(t, err, "failure creating user in setup")
		_, err = pgDB.AddUser(&updateTestUser2, nil)
		require.NoError(t, err, "failure creating user in setup")

		users, name, err := UpdateGroupAndMembers(ctx, testGroup.ID, "newName",
			[]model.UserID{updateTestUser1.ID}, []model.UserID{})
		require.NoError(t, err, "failed to update group")
		require.Equal(t, name, "newName", "group name not updated properly")
		index := usersContain(users, updateTestUser1.ID)
		require.NotEqual(t, -1, index, "group users not updated properly")

		users, name, err = UpdateGroupAndMembers(ctx, testGroup.ID, "anotherNewName",
			[]model.UserID{updateTestUser2.ID}, []model.UserID{updateTestUser1.ID})
		require.NoError(t, err, "failed to update group")
		require.Equal(t, name, "anotherNewName", "group name not updated properly")
		index = usersContain(users, updateTestUser1.ID)
		require.Equal(t, -1, index, "group users not removed properly")
		index = usersContain(users, updateTestUser2.ID)
		require.NotEqual(t, -1, index, "group users not updated properly")

		_, _, err = UpdateGroupAndMembers(ctx, testGroup.ID, "testGroup", nil, []model.UserID{-500})
		require.Error(t, err, "succeeded when update should have failed")
		group, err := GroupByIDTx(ctx, nil, testGroup.ID)
		require.NoError(t, err, "getting groups by ID failed")
		require.Equal(t, group.Name, "anotherNewName", "group name should not be updated")

		_, _, err = UpdateGroupAndMembers(ctx, testGroup.ID, "testGroup", []model.UserID{-500}, nil)
		require.Error(t, err, "succeeded when update should have failed")
		group, err = GroupByIDTx(ctx, nil, testGroup.ID)
		require.NoError(t, err, "getting groups by ID failed")
		require.Equal(t, group.Name, "anotherNewName", "group name should not be updated")

		users, name, err = UpdateGroupAndMembers(ctx, testGroup.ID, "testGroup", nil,
			[]model.UserID{updateTestUser2.ID, -500})
		require.NoError(t, err, "failed to update group")
		require.Equal(t, name, "testGroup", "group name not updated properly")
		require.GreaterOrEqual(t, 0, len(users), "group users not updated properly")
		index = usersContain(users, updateTestUser1.ID)
		require.Equal(t, -1, index, "group users not removed properly")
		index = usersContain(users, updateTestUser2.ID)
		require.Equal(t, -1, index, "group users not removed properly")
	})

	t.Run("test personal group", func(t *testing.T) {
		// Get personal group.
		groups, _, _, err := SearchGroups(ctx, "", testUser.ID, 0, 0)
		require.NoError(t, err)
		var personalGroup *Group
		for _, g := range groups {
			if g.OwnerID != 0 {
				require.Nil(t, personalGroup, "only one personal group should be returned")
				g := g
				personalGroup = &g
			}
		}
		require.NotNil(t, personalGroup, "no personal group returned")

		_, err = GroupByIDTx(ctx, nil, personalGroup.ID)
		require.ErrorIs(t, err, db.ErrNotFound)

		require.ErrorIs(t, DeleteGroup(ctx, personalGroup.ID), db.ErrNotFound)
		require.ErrorIs(t, UpdateGroupTx(ctx, nil, *personalGroup), db.ErrNotFound)
		require.ErrorIs(t, AddUsersToGroupTx(ctx, nil, personalGroup.ID), db.ErrNotFound)
		require.ErrorIs(t, RemoveUsersFromGroupTx(ctx, nil, personalGroup.ID), db.ErrNotFound)

		_, _, err = UpdateGroupAndMembers(ctx, personalGroup.ID, "", nil, nil)
		require.ErrorIs(t, err, db.ErrNotFound)

		// Personal group still returns no error for UsersInGroupTx.
		_, err = UsersInGroupTx(ctx, nil, personalGroup.ID)
		require.NoError(t, err)
	})
}

var (
	testGroup = Group{
		ID:   9001,
		Name: "testGroup",
	}
	testGroupStatic = Group{
		ID:   10001,
		Name: "testGroupStatic",
	}
	testGroups = []Group{testGroup, testGroupStatic}
	testUser   = model.User{
		ID:       1217651234,
		Username: fmt.Sprintf("IntegrationTest%d", 1217651234),
		Admin:    false,
		Active:   false,
	}
)

const (
	pathToMigrations = "file://../../static/migrations"
)

func setUp(ctx context.Context, t *testing.T, pgDB *db.PgDB) {
	_, err := pgDB.AddUser(&testUser, nil)
	require.NoError(t, err, "failure creating user in setup")

	_, _, err = AddGroupWithMembers(ctx, testGroupStatic, testUser.ID)
	require.NoError(t, err, "failure creating static test group")
}

func cleanUp(ctx context.Context, t *testing.T) {
	err := RemoveUsersFromGroupTx(ctx, nil, testGroup.ID, testUser.ID)
	if err != nil {
		t.Logf("Error cleaning up group membership: %v", err)
	}

	err = DeleteGroup(ctx, testGroup.ID)
	if err != nil {
		t.Logf("Error cleaning up group: %v", err)
	}

	err = DeleteGroup(ctx, testGroupStatic.ID)
	if err != nil {
		t.Logf("Error cleaning up static group: %v", err)
	}

	err = deleteUser(ctx, testUser.ID)
	if err != nil {
		t.Logf("Error cleaning up user: %v\n", err)
	}
}

// groupsContains returns -1 if group id was not found, else returns the index.
func groupsContain(groups []Group, id int) int {
	if len(groups) < 1 {
		return -1
	}

	for i, g := range groups {
		if g.ID == id {
			return i
		}
	}

	return -1
}

// usersContains returns -1 if user id was not found, else returns the index.
func usersContain(users []model.User, id model.UserID) int {
	if len(users) < 1 {
		return -1
	}

	for i, u := range users {
		if u.ID == id {
			return i
		}
	}

	return -1
}

func deleteUser(ctx context.Context, id model.UserID) error {
	_, err := db.Bun().NewDelete().Table("users").Where("id = ?", id).Exec(ctx)
	return err
}
