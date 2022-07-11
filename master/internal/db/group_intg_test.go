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

var (
	testGroup = model.Group{
		ID:   9001,
		Name: "kljhadsflkgjhjklsfhg",
	}
	testUser = model.User{
		ID:       1217651234,
		Username: fmt.Sprintf("IntegrationTest%d", 1217651234),
		Admin:    false,
		Active:   false,
	}
)

func TestHelloWorld(t *testing.T) {
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)

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
			pgDB.UpdateGroup(ctx, testGroup)
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
		err := pgDB.RemoveUsersFromGroup(ctx, testGroup.ID, testUser.ID)
		require.NoError(t, err, "failed to remove users from group")

		users, err := pgDB.GetUsersInGroup(ctx, testGroup.ID)
		require.NoError(t, err, "failed to look for users in group")

		i := usersContain(users, testUser.ID)
		require.Equal(t, -1, i, "User found in group after removing them from it")
	})

	t.Run("partial success on adding users to a group results in transaction rollback", func(t *testing.T) {
		err := pgDB.AddUsersToGroup(ctx, testGroup.ID, testUser.ID, 125674576, 12934728, 0, -15)
		// TODO: this being ErrNotFound in particular might be out of scope.
		// actual  : *pgconn.PgError(&pgconn.PgError{Severity:"ERROR", Code:"23503", Message:"insert or update on table \"user_group_membership\" violates foreign key constraint \"user_group_membership_user_id_fkey\"", Detail:"Key (user_id)=(125674576) is not present in table \"users\".", Hint:"", Position:0, InternalPosition:0, InternalQuery:"", Where:"", SchemaName:"public", TableName:"user_group_membership", ColumnName:"", DataTypeName:"", ConstraintName:"user_group_membership_user_id_fkey", File:"ri_triggers.c", Line:3266, Routine:"ri_ReportViolation"})
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
}

// TODO: test creating several groups and make sure it only deletes the one in question

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

	err = deleteUser(ctx, testUser.ID)
	if err != nil {
		t.Logf("Error cleaning up user: %v\n", err)
	}
}
