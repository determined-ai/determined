//go:build integration
// +build integration

package api

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestHelloWorld(t *testing.T) {
	ctx := context.Background()

	group, err := pgDB.AddGroup(ctx, model.Group{
		ID:   9001,
		Name: "kljhadsflkgjhjklsfhg",
	})
	failNowIfErrOtherThanAlreadyExists(t, err)

	// Clean up after ourselves in case there's a failure along the way
	defer func(id int) {
		err := pgDB.DeleteGroup(ctx, id)
		if err != nil {
			t.Logf("Error cleaning up after ourselves: %v", err)
		}
	}(group.ID)

	groups, err := pgDB.SearchGroups(ctx, 0)
	failNowIfErr(t, err)

	index := groupsContain(groups, group.ID)
	if index == -1 {
		t.Fatalf("Expected groups to contain the new one")
	}

	foundGroup := groups[index]
	if foundGroup.Name != group.Name {
		t.Fatalf("Expected found group to have the same name ('%s' vs '%s')", foundGroup.Name, group.Name)
	}

	foundGroup, err = pgDB.GroupByID(ctx, group.ID)
	failNowIfErr(t, err)
	if foundGroup.Name != group.Name {
		t.Fatalf("Expected group found by id to have the same name")
	}

	newName := "kljhadsflkgjhjklsfhgasdhj"
	group.Name = newName
	err = pgDB.UpdateGroup(ctx, group)
	failNowIfErr(t, err)

	foundGroup, err = pgDB.GroupByID(ctx, group.ID)
	failNowIfErrOtherThanAlreadyExists(t, err)
	if foundGroup.Name != newName {
		t.Fatalf("Expected updated group to have new name")
	}

	var userID model.UserID = 1217651234
	newUser := model.User{
		ID:       userID,
		Username: fmt.Sprintf("IntegrationTest%d", userID),
		Admin:    false,
		Active:   false,
	}
	_, err = pgDB.AddUser(&newUser, nil)
	failNowIfErrOtherThanAlreadyExists(t, err)

	err = pgDB.AddUsersToGroup(ctx, group.ID, newUser.ID)
	failNowIfErrOtherThanAlreadyExists(t, err)
	defer func(groupID int, userID model.UserID) {
		err = pgDB.RemoveUsersFromGroup(ctx, groupID, newUser.ID)
	}(group.ID, newUser.ID)

	users, err := pgDB.GetUsersInGroup(ctx, group.ID)
	failNowIfErr(t, err)

	index = usersContain(users, newUser.ID)
	if index == -1 {
		t.Fatal("Expected to find user in group we added them to")
	}

	groups, err = pgDB.SearchGroups(ctx, newUser.ID)
	failNowIfErr(t, err)
	if i := groupsContain(groups, group.ID); i == -1 {
		t.Fatalf("Expected to find group that our user belongs to")
	}

	err = pgDB.RemoveUsersFromGroup(ctx, group.ID, newUser.ID)
	failNowIfErr(t, err)

	users, err = pgDB.GetUsersInGroup(ctx, group.ID)
	failNowIfErr(t, err)
	if i := usersContain(users, newUser.ID); i != -1 {
		t.Fatal("Expected not to find user in group after removing them")
	}
}

func failNowIfErr(t *testing.T, err error) {
	if err != nil {
		panic(fmt.Sprintf("Failed because of an error: %s", err.Error()))
	}
}

func failNowIfErrOtherThanAlreadyExists(t *testing.T, err error) {
	if errAlreadyExists(err) {
		return
	}

	failNowIfErr(t, err)
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

	if err == db.ErrDuplicateRecord {
		return true
	}

	msg := err.Error()

	// FIXME: hack! Try to find another way to detect this
	if strings.Contains(msg, "duplicate key value violates unique constraint") {
		return true
	}

	return false
}
