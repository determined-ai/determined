//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestUserGroups(t *testing.T) {
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, migrationsFromDB)

	t.Cleanup(func() { cleanUp(ctx, t, pgDB) })
	setUp(ctx, t, pgDB)

	t.Run("group creation", func(t *testing.T) {
		_, err := pgDB.AddGroup(ctx, testGroup)
		require.NoError(t, err, "failed to create group")
	})

	t.Run("search groups", func(t *testing.T) {
		groups, err := pgDB.SearchGroups(ctx, 0)
		require.NoError(t, err, "failed to search for groups")

		index := groupsContain(groups, testGroup.ID)
		foundGroup := groups[index]
		require.NotEqual(t, -1, index, "Expected groups to contain the new one")
		require.Equal(t, testGroup.Name, foundGroup.Name, "Expected found group to have the same name as the one we created")
	})

	t.Run("find group by id", func(t *testing.T) {
		foundGroup, err := pgDB.GroupByID(ctx, testGroup.ID)
		require.NoError(t, err, "failed to find group by id")
		require.Equal(t, testGroup.Name, foundGroup.Name, "Expected found group to have the same name as the one we created")
	})

	t.Run("update group", func(t *testing.T) {
		// Put it back the way it was when we're done
		defer func(name string) {
			testGroup.Name = name
			err := pgDB.UpdateGroup(ctx, testGroup)
			require.NoError(t, err, "failed to put things back how they were after testing UpdateGroup")
		}(testGroup.Name)

		newName := "kljhadsflkgjhjklsfhgasdhj"
		testGroup.Name = newName
		err := pgDB.UpdateGroup(ctx, testGroup)
		require.NoError(t, err, "failed to update group")

		foundGroup, err := pgDB.GroupByID(ctx, testGroup.ID)
		require.NoError(t, err, "failed to find group by id")
		require.Equal(t, newName, foundGroup.Name, "Expected found group to have the new name")
	})

	t.Run("add users to group", func(t *testing.T) {
		err := pgDB.AddUsersToGroup(ctx, testGroup.ID, testUser.ID)
		require.NoError(t, err, "failed to add users to group")

		users, err := pgDB.GetUsersInGroup(ctx, testGroup.ID)
		require.NoError(t, err, "failed to search for users that belong to group")

		index := usersContain(users, testUser.ID)
		require.NotEqual(t, -1, index, "Expected users in group to contain the newly added one")
	})

	t.Run("search groups by user membership", func(t *testing.T) {
		groups, err := pgDB.SearchGroups(ctx, testUser.ID)
		require.NoError(t, err, "failed to search for groups that user blongs to")

		index := groupsContain(groups, testGroup.ID)
		require.NotEqual(t, -1, index, "Group user was added to not found when searching by user membership")
	})

	t.Run("remove users from group", func(t *testing.T) {
		err := pgDB.RemoveUsersFromGroup(ctx, testGroup.ID, -500)
		require.Equal(t, ErrNotFound, err, "failed to return ErrNotFound when removing non-existent users from group")

		err = pgDB.RemoveUsersFromGroup(ctx, testGroup.ID, testGroup.UserID, -500)
		require.Equal(t, ErrNotFound, err, "failed to return ErrNotFound when trying to remove a mix of users in a group and not")

		err = pgDB.RemoveUsersFromGroup(ctx, testGroup.ID, testUser.ID)
		require.NoError(t, err, "failed to remove users from group")

		users, err := pgDB.GetUsersInGroup(ctx, testGroup.ID)
		require.NoError(t, err, "failed to look for users in group")

		i := usersContain(users, testUser.ID)
		require.Equal(t, -1, i, "User found in group after removing them from it")

		err = pgDB.RemoveUsersFromGroup(ctx, testGroup.ID, testUser.ID)
		require.Equal(t, ErrNotFound, err, "failed to return ErrNotFound when trying to remove users from group they're not in")
	})

	t.Run("partial success on adding users to a group results in tx rollback and ErrNotFound", func(t *testing.T) {
		err := pgDB.AddUsersToGroup(ctx, testGroup.ID, testUser.ID, 125674576, 12934728, 0, -15)
		require.Equal(t, ErrNotFound, err, "didn't return ErrNotFound when adding non-existent users to a group")

		users, err := pgDB.GetUsersInGroup(ctx, testGroup.ID)
		require.NoError(t, err, "failed to search for users that belong to group")

		index := usersContain(users, testUser.ID)
		require.Equal(t, -1, index, "Expected users in group not to contain the one added in the erroring call")
	})

	t.Run("AddUsersToGroup fails with ErrNotFound when attempting to add users to a non-existent group", func(t *testing.T) {
		err := pgDB.AddUsersToGroup(ctx, -500, testUser.ID)
		require.Equal(t, ErrNotFound, err, "didn't return ErrNotFound when trying to add users to a non-existent group")
	})

	t.Run("Deleting a group that doesn't exist results in ErrNotFound", func(t *testing.T) {
		err := pgDB.DeleteGroup(ctx, -500)
		require.Equal(t, ErrNotFound, err, "didn't return ErrNotFound when trying to delete a non-existent group")
	})

	t.Run("Deleting a group that has users should work", func(t *testing.T) {
		tmpGroup := testGroup
		tmpGroup.ID++
		tmpGroup.Name += tmpGroup.Name

		_, err := pgDB.AddGroup(ctx, tmpGroup)
		require.NoError(t, err, "failed to create group")

		err = pgDB.AddUsersToGroup(ctx, tmpGroup.ID, testUser.ID)
		require.NoError(t, err, "failed to add user to group")

		err = pgDB.DeleteGroup(ctx, tmpGroup.ID)
		require.NoError(t, err, "errored when deleting group")

		_, err = pgDB.GroupByID(ctx, tmpGroup.ID)
		require.Equal(t, ErrNotFound, err, "deleted group should not be found, and ErrNotFound returned")
	})

	t.Run("AddGroup returns ErrDuplicateRecord when creating a group that already exists", func(t *testing.T) {
		_, err := pgDB.AddGroup(ctx, testGroupStatic)
		require.Equal(t, ErrDuplicateRecord, err, "didn't return ErrDuplicateRecord")
	})

	t.Run("AddUsersToGroup returns ErrDuplicateRecord when adding users to a group they're already in", func(t *testing.T) {
		err := pgDB.AddUsersToGroup(ctx, testGroupStatic.ID, testUser.ID)
		require.Equal(t, ErrDuplicateRecord, err, "should have returned ErrDuplicateRecord")
	})

	t.Run("Static test group should exist at the end and test user should be in it", func(t *testing.T) {
		_, err := pgDB.GroupByID(ctx, testGroupStatic.ID)
		require.NoError(t, err, "errored while getting static test group")

		users, err := pgDB.GetUsersInGroup(ctx, testGroupStatic.ID)
		require.NoError(t, err, "failed to search for users that belong to static group")

		index := usersContain(users, testUser.ID)
		require.NotEqual(t, -1, index, "Expected users in static group to contain the test user")
	})
}

var (
	testGroup = model.Group{
		ID:   9001,
		Name: "kljhadsflkgjhjklsfhg",
	}
	testGroupStatic = model.Group{
		ID:   10001,
		Name: "dsjkfkljjkasdfasdky",
	}
	testUser = model.User{
		ID:       1217651234,
		Username: fmt.Sprintf("IntegrationTest%d", 1217651234),
		Admin:    false,
		Active:   false,
	}
)

// groupsContains returns -1 if group id was not found, else returns the index
func groupsContain(groups []model.Group, id int) int {
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
	_, err := Bun().NewDelete().Table("users").Where("id = ?", id).Exec(ctx)
	return err
}

func setUp(ctx context.Context, t *testing.T, pgDB *PgDB) {
	_, err := pgDB.AddUser(&testUser, nil)
	require.NoError(t, err, "failure creating user in setup")

	_, err = pgDB.AddGroup(ctx, testGroupStatic)
	require.NoError(t, err, "failure creating static test group")

	err = pgDB.AddUsersToGroup(ctx, testGroupStatic.ID, testUser.ID)
	require.NoError(t, err, "failure adding user to static group")
}

func cleanUp(ctx context.Context, t *testing.T, pgDB *PgDB) {
	err := pgDB.RemoveUsersFromGroup(ctx, testGroup.ID, testUser.ID)
	if err != nil {
		t.Logf("Error cleaning up group membership: %v", err)
	}

	err = pgDB.DeleteGroup(ctx, testGroup.ID)
	if err != nil {
		t.Logf("Error cleaning up group: %v", err)
	}

	err = pgDB.DeleteGroup(ctx, testGroupStatic.ID)
	if err != nil {
		t.Logf("Error cleaning up static group: %v", err)
	}

	err = deleteUser(ctx, testUser.ID)
	if err != nil {
		t.Logf("Error cleaning up user: %v\n", err)
	}
}
