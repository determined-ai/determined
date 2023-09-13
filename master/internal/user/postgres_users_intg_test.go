//go:build integration
// +build integration

package user

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/schema"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestMain(m *testing.M) {
	pgDB, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = pgDB.Migrate("file://../../static/migrations", []string{"up"})
	if err != nil {
		log.Panicln(err)
	}
	err = db.InitAuthKeys()
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

// Tests for postgres_users.go.
func TestUserAdd(t *testing.T) {
	cases := []struct {
		name string
		user model.User
		ug   model.AgentUserGroup
	}{
		{
			"simple-case",
			model.User{Username: uuid.NewString()},
			model.AgentUserGroup{},
		},
		{
			"aug-defined",
			model.User{Username: uuid.NewString()},
			model.AgentUserGroup{User: uuid.NewString(), UID: 1},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Test Add.
			uid, err := Add(context.TODO(), &tt.user, &tt.ug)
			require.NoError(t, err)

			// require entry in the users table (addUser).
			var resUser model.User

			err = db.Bun().NewSelect().Table("users").Where("id = ?", uid).Scan(context.TODO(), &resUser)
			require.NoError(t, err)
			// Reset the modified at variable, just for testing purposes.
			resUser.ModifiedAt = tt.user.ModifiedAt
			require.Equal(t, tt.user, resUser)

			// require entry in the group table (addUser).
			var resGroup model.Group
			err = db.Bun().NewSelect().Table("groups").
				Where("user_id = ?", uid).Scan(context.TODO(), &resGroup)
			require.NoError(t, err)
			require.Equal(t, uid, resGroup.OwnerID)
			require.Equal(t, fmt.Sprintf("%d%s", uid, PersonalGroupPostfix), resGroup.Name)

			// require entry in the group membership table (addUser).
			var resGroupMem model.GroupMembership
			err = db.Bun().NewSelect().Table("user_group_membership").
				Where("user_id = ?", uid).Scan(context.TODO(), &resGroupMem)
			require.NoError(t, err)
			require.Equal(t, uid, resGroupMem.UserID)
			require.Equal(t, resGroup.ID, resGroupMem.GroupID)

			// require entry in the AUG membership table (addAgentUserGroup).
			// But also check that the AUG isn't nil/empty.
			var resAUG model.AgentUserGroup
			if tt.ug != (model.AgentUserGroup{}) {
				err = db.Bun().NewSelect().Table("agent_user_groups").
					Where("user_id = ?", uid).Scan(context.TODO(), &resAUG)
				require.NoError(t, err)
				require.Equal(t, uid, resAUG.UserID)
				require.Equal(t, tt.ug.User, resAUG.User)
				require.Equal(t, tt.ug.UID, resAUG.UID)
			}
		})
	}
}

func TestUserUpdate(t *testing.T) {
	cases := []struct {
		name        string
		ug          model.AgentUserGroup
		updatedUser model.User
	}{
		{
			"simple-case",
			model.AgentUserGroup{},
			model.User{Username: uuid.NewString()},
		},
		{
			"aug-defined-1",
			model.AgentUserGroup{User: uuid.NewString(), UID: 1},
			model.User{Username: uuid.NewString(), Admin: false, Active: false, Remote: false},
		},
		{
			"aug-defined-2",
			model.AgentUserGroup{User: uuid.NewString(), UID: 123},
			model.User{Username: uuid.NewString(), Admin: true, Active: true, Remote: true},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			user, err := addTestUser(nil)
			require.NoError(t, err)

			// Test Update.
			tt.updatedUser.ID = user.ID
			err = Update(context.TODO(), &tt.updatedUser, []string{"admin", "active", "remote"}, &tt.ug)
			require.NoError(t, err)

			// Now require that the fields are indeed updated (except DisplayName).
			var updatedUser model.User
			err = db.Bun().NewSelect().Table("users").
				Where("id = ?", tt.updatedUser.ID).Scan(context.TODO(), &updatedUser)
			require.NoError(t, err)
			// Check for update.
			require.Equal(t, tt.updatedUser.Admin, updatedUser.Admin)
			require.Equal(t, tt.updatedUser.Active, updatedUser.Active)
			require.Equal(t, tt.updatedUser.Remote, updatedUser.Remote)
			// Check that it HASN'T been updated.
			require.Equal(t, user.DisplayName, updatedUser.DisplayName)
			require.Equal(t, user.Username, updatedUser.Username)
		})
	}
}

func TestUserStartSession(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a session.
	token, err := StartSession(context.TODO(), user)
	require.NoError(t, err)
	require.NotNil(t, token)

	exists, err := db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", user.ID).Exists(context.TODO())
	require.True(t, exists)
	require.NoError(t, err)
}

func TestDeleteSessionByToken(t *testing.T) {
	userID, _, token, err := addTestSession()
	require.NoError(t, err)

	err = DeleteSessionByToken(context.TODO(), token)
	require.NoError(t, err)

	exists, err := db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", userID).Exists(context.TODO())
	require.False(t, exists)
	require.ErrorAs(t, errors.New("Receive unexpected error: bun: Model(nil)"), &err)
}

func TestDeleteSessionByID(t *testing.T) {
	userID, sessionID, _, err := addTestSession()
	require.NoError(t, err)

	err = DeleteSessionByID(context.TODO(), sessionID)
	require.NoError(t, err)

	exists, err := db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", userID).Exists(context.TODO())
	require.False(t, exists)
	require.ErrorAs(t, errors.New("Receive unexpected error: bun: Model(nil)"), &err)
}

func TestUpdateUsername(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	newUsername := uuid.NewString()
	err = UpdateUsername(context.TODO(), &user.ID, newUsername)
	require.NoError(t, err)

	// Now require that the username is indeed updated.
	var resUser model.User
	err = db.Bun().NewSelect().
		Table("users").Where("id = ?", user.ID).
		Scan(context.TODO(), &resUser)
	require.NoError(t, err)
	require.Equal(t, newUsername, resUser.Username)
}

func TestList(t *testing.T) {
	initialValues, err := List(context.TODO())
	require.NoError(t, err)

	_, err = addTestUser(nil)
	require.NoError(t, err)

	newValues, err := List(context.TODO())
	require.NoError(t, err)
	require.Equal(t, 1, len(newValues)-len(initialValues))
}

func TestByID(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	fullUser, err := ByID(context.TODO(), user.ID)
	require.NoError(t, err)
	require.Equal(t, fullUser.ID, user.ID)
}

func TestByToken(t *testing.T) {
	userID, sessionID, token, err := addTestSession()
	require.NoError(t, err)

	user, session, err := ByToken(context.TODO(), token, &model.ExternalSessions{})
	require.NoError(t, err)
	require.Equal(t, user.ID, userID)
	require.Equal(t, session.ID, sessionID)
}

func TestByUsername(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	u, err := ByUsername(context.TODO(), user.Username)
	require.NoError(t, err)
	require.Equal(t, u, user)
}

// Tests for postgres_agentusergroup.go.
func TestGetAgentUserGroup(t *testing.T) {
	cases := []struct {
		name         string
		userAug      *model.AgentUserGroup
		workspaceAug *model.AgentUserGroup
	}{
		{"simple-case", nil, nil},
		{"exp-defined-0", &model.AgentUserGroup{}, &model.AgentUserGroup{}},
		{
			"exp-defined-1",
			&model.AgentUserGroup{UID: 10, User: "test-10", GID: 100, Group: "group-10"},
			&model.AgentUserGroup{UID: 1, User: "test-1", GID: 11, Group: "group-1"},
		},
		{
			"exp-defined-2",
			&model.AgentUserGroup{},
			&model.AgentUserGroup{UID: 2, User: "test-2", GID: 22, Group: "group-2"},
		},
		{
			"exp-defined-3",
			&model.AgentUserGroup{UID: 3, User: "test-3", GID: 33, Group: "group-3"},
			&model.AgentUserGroup{},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Add a user & its own agent user group
			user, err := addTestUser(tt.userAug)
			require.NoError(t, err)

			// Add a workspace
			workspaceID, err := addTestWorkspace(user.ID, tt.workspaceAug)
			require.NoError(t, err)

			aug, err := GetAgentUserGroup(user.ID, workspaceID)
			require.NoError(t, err)
			if tt.workspaceAug == nil {
				nilAug := model.AgentUserGroup{
					BaseModel: schema.BaseModel{},
					ID:        0, UserID: 0,
					User: "root", UID: 0,
					Group: "root", GID: 0,
					RelatedUser: (*model.User)(nil),
				}
				require.Equal(t, &nilAug, aug)
			}
			require.NotNil(t, aug)
			if tt.workspaceAug != nil {
				require.Equal(t, *tt.workspaceAug, *aug)
				return
			}
			if tt.userAug != nil {
				require.Equal(t, *tt.userAug, *aug)
				return
			}
		})
	}
}

func addTestWorkspace(userID model.UserID, aug *model.AgentUserGroup) (int, error) {
	wksp := model.Workspace{
		Name:   uuid.NewString(),
		UserID: userID,
	}
	if aug != nil {
		uid := int32(aug.UID)
		gid := int32(aug.GID)
		wksp.AgentUID = &uid
		wksp.AgentUser = &aug.User
		wksp.AgentGID = &gid
		wksp.AgentGroup = &aug.Group
	}

	_, err := db.Bun().NewInsert().Model(&wksp).Exec(context.TODO())
	if err != nil {
		return 0, err
	}
	var res model.Workspace
	err = db.Bun().NewSelect().Model(&res).
		Where("name = ?", wksp.Name).Scan(context.TODO())
	if err != nil {
		return 0, err
	}

	return res.ID, nil
}

func addTestSession() (model.UserID, model.SessionID, string, error) {
	// Add a user.
	user, err := addTestUser(nil)
	if err != nil {
		return 0, 0, "", err
	}

	// Add a session.
	var session model.UserSession
	token, err := StartSession(context.TODO(), user)
	if err != nil {
		return 0, 0, "", fmt.Errorf("couldn't create new session: %w", err)
	}

	if err = db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", user.ID).Scan(context.TODO(), &session); err != nil {
		return 0, 0, "", fmt.Errorf("couldn't create new session: %w", err)
	}
	return user.ID, session.ID, token, nil
}

func addTestUser(aug *model.AgentUserGroup) (*model.User, error) {
	var user model.User
	uid, err := Add(context.TODO(), &model.User{Username: uuid.NewString()}, aug)
	if err != nil {
		return nil, fmt.Errorf("couldn't create new user: %w", err)
	}
	err = db.Bun().NewSelect().Table("users").Where("id = ?", uid).Scan(context.TODO(), &user)
	return &user, err
}
