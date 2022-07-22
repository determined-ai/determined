//go:build integration
// +build integration

package usergroup

import (
	"context"
	"fmt"
	"testing"

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
		_, err := AddGroup(ctx, testGroup)
		require.NoError(t, err, "failed to create group")
	})

	t.Run("search groups", func(t *testing.T) {
		groups, err := SearchGroups(ctx, "", 0, 0, 0)
		require.NoError(t, err, "failed to search for groups")

		index := groupsContain(groups, testGroup.ID)
		require.NotEqual(t, -1, index, "Expected groups to contain the new one")
		foundGroup := groups[index]
		require.Equal(t, testGroup.Name, foundGroup.Name, "Expected found group to have the same name as the one we created")

		groups, err = SearchGroups(ctx, testGroup.Name, 0, 0, 0)
		require.NoError(t, err, "failed to search for groups")
		require.NotEmpty(t, groups, "failed to find group by name")
		require.Len(t, groups, 1, "failed to narrow search to just matching name")
		require.Equal(t, testGroup.Name, groups[0].Name, "failed to find the correct group")
	})

	t.Run("find group by id", func(t *testing.T) {
		foundGroup, err := GroupByID(ctx, testGroup.ID)
		require.NoError(t, err, "failed to find group by id")
		require.Equal(t, testGroup.Name, foundGroup.Name, "Expected found group to have the same name as the one we created")
	})

	t.Run("update group", func(t *testing.T) {
		// Put it back the way it was when we're done
		defer func(name string) {
			testGroup.Name = name
			err := UpdateGroup(ctx, testGroup)
			require.NoError(t, err, "failed to put things back how they were after testing UpdateGroup")
		}(testGroup.Name)

		newName := "kljhadsflkgjhjklsfhgasdhj"
		testGroup.Name = newName
		err := UpdateGroup(ctx, testGroup)
		require.NoError(t, err, "failed to update group")

		foundGroup, err := GroupByID(ctx, testGroup.ID)
		require.NoError(t, err, "failed to find group by id")
		require.Equal(t, newName, foundGroup.Name, "Expected found group to have the new name")
	})

	t.Run("add users to group", func(t *testing.T) {
		err := AddUsersToGroup(ctx, testGroup.ID, testUser.ID)
		require.NoError(t, err, "failed to add users to group")

		users, err := UsersInGroup(ctx, testGroup.ID)
		require.NoError(t, err, "failed to search for users that belong to group")
		require.Len(t, users, 1, "failed to return only the set of users in the group")

		index := usersContain(users, testUser.ID)
		require.NotEqual(t, -1, index, "Expected users in group to contain the newly added one")
	})

	t.Run("search groups by user membership", func(t *testing.T) {
		groups, err := SearchGroups(ctx, "", testUser.ID, 0, 0)
		require.NoError(t, err, "failed to search for groups that user blongs to")

		index := groupsContain(groups, testGroup.ID)
		require.NotEqual(t, -1, index, "Group user was added to not found when searching by user membership")
	})

	t.Run("remove users from group", func(t *testing.T) {
		err := RemoveUsersFromGroup(ctx, testGroup.ID, -500)
		require.Equal(t, db.ErrNotFound, err, "failed to return ErrNotFound when removing non-existent users from group")

		err = RemoveUsersFromGroup(ctx, testGroup.ID, testUser.ID, -500)
		require.NoError(t, err, "erroneously returned error when trying to remove a mix of users in a group and not")

		users, err := UsersInGroup(ctx, testGroup.ID)
		require.NoError(t, err, "failed to look for users in group")

		i := usersContain(users, testUser.ID)
		require.Equal(t, -1, i, "User found in group after removing them from it")

		err = RemoveUsersFromGroup(ctx, testGroup.ID, testUser.ID)
		require.Equal(t, db.ErrNotFound, err, "failed to return ErrNotFound when trying to remove users from group they're not in")
	})

	t.Run("partial success on adding users to a group results in tx rollback and ErrNotFound", func(t *testing.T) {
		err := AddUsersToGroup(ctx, testGroup.ID, testUser.ID, 125674576, 12934728, 0, -15)
		require.Equal(t, db.ErrNotFound, err, "didn't return ErrNotFound when adding non-existent users to a group")

		users, err := UsersInGroup(ctx, testGroup.ID)
		require.NoError(t, err, "failed to search for users that belong to group")

		index := usersContain(users, testUser.ID)
		require.Equal(t, -1, index, "Expected users in group not to contain the one added in the erroring call")
	})

	t.Run("AddUsersToGroup fails with ErrNotFound when attempting to add users to a non-existent group", func(t *testing.T) {
		err := AddUsersToGroup(ctx, -500, testUser.ID)
		require.Equal(t, db.ErrNotFound, err, "didn't return ErrNotFound when trying to add users to a non-existent group")
	})

	t.Run("Deleting a group that doesn't exist results in ErrNotFound", func(t *testing.T) {
		err := DeleteGroup(ctx, -500)
		require.Equal(t, db.ErrNotFound, err, "didn't return ErrNotFound when trying to delete a non-existent group")
	})

	t.Run("Deleting a group that has users should work", func(t *testing.T) {
		tmpGroup := testGroup
		tmpGroup.ID++
		tmpGroup.Name += tmpGroup.Name

		_, err := AddGroup(ctx, tmpGroup)
		require.NoError(t, err, "failed to create group")

		err = AddUsersToGroup(ctx, tmpGroup.ID, testUser.ID)
		require.NoError(t, err, "failed to add user to group")

		err = DeleteGroup(ctx, tmpGroup.ID)
		require.NoError(t, err, "errored when deleting group")

		_, err = GroupByID(ctx, tmpGroup.ID)
		require.Equal(t, db.ErrNotFound, err, "deleted group should not be found, and ErrNotFound returned")
	})

	t.Run("AddGroup returns ErrDuplicateRecord when creating a group that already exists", func(t *testing.T) {
		_, err := AddGroup(ctx, testGroupStatic)
		require.Equal(t, db.ErrDuplicateRecord, err, "didn't return ErrDuplicateRecord")
	})

	t.Run("AddUsersToGroup returns ErrDuplicateRecord when adding users to a group they're already in", func(t *testing.T) {
		err := AddUsersToGroup(ctx, testGroupStatic.ID, testUser.ID)
		require.Equal(t, db.ErrDuplicateRecord, err, "should have returned ErrDuplicateRecord")
	})

	t.Run("Static test group should exist at the end and test user should be in it", func(t *testing.T) {
		_, err := GroupByID(ctx, testGroupStatic.ID)
		require.NoError(t, err, "errored while getting static test group")

		users, err := UsersInGroup(ctx, testGroupStatic.ID)
		require.NoError(t, err, "failed to search for users that belong to static group")

		index := usersContain(users, testUser.ID)
		require.NotEqual(t, -1, index, "Expected users in static group to contain the test user")
	})

	t.Run("search group with offsets and limits", func(t *testing.T) {
		answerGroups, err := SearchGroups(ctx, "", 0, 0, 3)
		require.NoError(t, err, "failed to search for groups")

		groups, err := SearchGroups(ctx, "", 0, 0, 1)
		require.NoError(t, err, "failed to search for groups")
		index := groupsContain(groups, answerGroups[0].ID)
		require.NotEqual(t, -1, index, "Expected groups to contain the new one")
		foundGroup := groups[index]
		require.Equal(t, answerGroups[0].Name, foundGroup.Name, "Expected found group to have the same name as the first answerGroup")
		require.Equal(t, 1, len(groups), "Expected no more than one group to have been returned")

		groups, err = SearchGroups(ctx, "", 0, 1, 2)
		require.NoError(t, err, "failed to search for groups")
		require.Equal(t, 2, len(groups), "Expected no more than two groups to have been returned")
		index = groupsContain(groups, answerGroups[1].ID)
		require.NotEqual(t, -1, index, "Expected groups to contain the new one")
		foundGroup = groups[index]
		require.Equal(t, answerGroups[1].Name, foundGroup.Name, "Expected found group to have the same name as the second answerGroup")

		index = groupsContain(groups, answerGroups[2].ID)
		require.NotEqual(t, -1, index, "Expected groups to contain the new one")
		foundGroup = groups[index]
		require.Equal(t, answerGroups[2].Name, foundGroup.Name, "Expected found group to have the same name as the third answerGroup")
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
	testUser = model.User{
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

	_, err = AddGroup(ctx, testGroupStatic)
	require.NoError(t, err, "failure creating static test group")

	err = AddUsersToGroup(ctx, testGroupStatic.ID, testUser.ID)
	require.NoError(t, err, "failure adding user to static group")
}

func cleanUp(ctx context.Context, t *testing.T) {
	err := RemoveUsersFromGroup(ctx, testGroup.ID, testUser.ID)
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

// groupsContains returns -1 if group id was not found, else returns the index
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

// usersContains returns -1 if user id was not found, else returns the index
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
