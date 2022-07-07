//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"strings"
	"testing"

	// "github.com/determined-ai/determined/master/internal/db"
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
	db := MustResolveTestPostgres(t)

	t.Cleanup(func() { cleanUp(ctx, t) })
	setUp(ctx, t)

	t.Run("group creation", func(t *testing.T) {
		_, err := pgDB.AddGroup(ctx, testGroup)
		failNowIfErr(t, err)
	})

	t.Run("search groups", func(t *testing.T) {
		groups, err := pgDB.SearchGroups(ctx, 0)
		failNowIfErr(t, err)

		index := groupsContain(groups, testGroup.ID)
		if index == -1 {
			t.Fatalf("Expected groups to contain the new one")
		}
		foundGroup := groups[index]
		if foundGroup.Name != testGroup.Name {
			t.Fatalf("Expected found group to have the same name ('%s' vs '%s')", foundGroup.Name, testGroup.Name)
		}
	})

	t.Run("find group by id", func(t *testing.T) {
		foundGroup, err := pgDB.GroupByID(ctx, testGroup.ID)
		failNowIfErr(t, err)
		if foundGroup.Name != testGroup.Name {
			t.Fatalf("Expected group found by id to have the same name")
		}
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
		failNowIfErr(t, err)

		foundGroup, err := pgDB.GroupByID(ctx, testGroup.ID)
		failNowIfErr(t, err)
		if foundGroup.Name != newName {
			t.Fatalf("Expected updated group to have new name")
		}
	})

	t.Run("add users to groups", func(t *testing.T) {
		err := pgDB.AddUsersToGroup(ctx, testGroup.ID, testUser.ID)
		failNowIfErr(t, err)
		// Clean up
		defer func(groupID int, userID model.UserID) {

		}(testGroup.ID, testUser.ID)

		users, err := pgDB.GetUsersInGroup(ctx, testGroup.ID)
		failNowIfErr(t, err)

		index := usersContain(users, testUser.ID)
		if index == -1 {
			t.Fatal("Expected to find user in group we added them to")
		}
	})

	t.Run("search groups by user membership", func(t *testing.T) {
		groups, err := pgDB.SearchGroups(ctx, testUser.ID)
		failNowIfErr(t, err)

		if i := groupsContain(groups, testGroup.ID); i == -1 {
			t.Fatalf("Expected to find group that our user belongs to")
		}
	})

	t.Run("remove users from group", func(t *testing.T) {
		err := pgDB.RemoveUsersFromGroup(ctx, testGroup.ID, testUser.ID)
		failNowIfErr(t, err)

		users, err := pgDB.GetUsersInGroup(ctx, testGroup.ID)
		failNowIfErr(t, err)

		if i := usersContain(users, testUser.ID); i != -1 {
			t.Fatal("Expected not to find user in group after removing them")
		}
	})
}

// TODO: test creating several groups and make sure it only deletes the one in question
// TODO: move to master/internal/db, where other integration tests are

func failNowIfErr(t *testing.T, err error) {
	if err != nil {
		panic(fmt.Sprintf("Failed because of an error: %s", err.Error()))
	}
}

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

func errAlreadyExists(err error) bool {
	if err == nil {
		return false
	}

	if err == ErrDuplicateRecord {
		return true
	}

	msg := err.Error()

	// FIXME: hack! Try to find another way to detect this
	if strings.Contains(msg, "duplicate key value violates unique constraint") {
		return true
	}

	return false
}

func deleteUser(ctx context.Context, id model.UserID) error {
	_, err := Bun().NewDelete().Table("users").Where("id = ?", id).Exec(ctx)
	return err
}

func setUp(ctx context.Context, t *testing.T) {
	_, err := pgDB.AddUser(&testUser, nil)
	failNowIfErr(t, err)
}

func cleanUp(ctx context.Context, t *testing.T) {
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
