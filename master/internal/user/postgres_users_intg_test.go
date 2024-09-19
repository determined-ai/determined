//go:build integration
// +build integration

package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/schema"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const desc = "test desc"

func TestMain(m *testing.M) {
	pgDB, _, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
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
			resUser.LastAuthAt = tt.user.LastAuthAt
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

func TestUserAddDuplicate(t *testing.T) {
	username := uuid.NewString()
	user1 := model.User{Username: username}
	user2 := model.User{Username: username}

	// Test Add.
	_, err := Add(context.TODO(), &user1, &model.AgentUserGroup{})
	require.NoError(t, err)

	// Then try to add another user with the same username, expect an error.
	_, err = Add(context.TODO(), &user2, &model.AgentUserGroup{})
	require.Equal(t, err, db.ErrDuplicateRecord)

	if pgerr, ok := errors.Cause(err).(*pgconn.PgError); ok {
		require.Equal(t, db.CodeUniqueViolation, pgerr.Code)
	}
}

func TestUserUpdate(t *testing.T) {
	cases := []struct {
		name        string
		ug          *model.AgentUserGroup
		toUpdate    []string
		updatedUser model.User
	}{
		{
			"simple-case",
			&model.AgentUserGroup{},
			[]string{"admin", "active", "remote"},
			model.User{Username: uuid.NewString()},
		},
		{
			"aug defined and user is not admin, inactive, not remote",
			&model.AgentUserGroup{User: uuid.NewString(), UID: 1},
			[]string{"admin", "active", "remote"},
			model.User{Username: uuid.NewString(), Admin: false, Active: false, Remote: false},
		},
		{
			"aug defined and user is admin, active and remote",
			&model.AgentUserGroup{User: uuid.NewString(), UID: 123},
			[]string{"admin", "active", "remote"},
			model.User{Username: uuid.NewString(), Admin: true, Active: true, Remote: true},
		},
		{
			"update password",
			nil,
			[]string{"password_hash"},
			model.User{Username: uuid.NewString(), PasswordHash: null.NewString(uuid.NewString(), true)},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			user, err := addTestUser(nil)
			require.NoError(t, err)

			// Test Update.
			tt.updatedUser.ID = user.ID
			err = Update(context.TODO(), &tt.updatedUser, tt.toUpdate, tt.ug)
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
		Where("user_id = ?", user.ID).
		Where("token_type = ?", model.TokenTypeUserSession).
		Exists(context.TODO())
	require.True(t, exists)
	require.NoError(t, err)
}

func TestUserStartSessionTokenHasClaims(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a session with inherited claims.
	claims := map[string]string{"test_key": "test_val"}
	token, err := StartSession(context.TODO(), user, WithInheritedClaims(claims))
	require.NoError(t, err)
	require.NotNil(t, token)

	var restoredSession model.UserSession
	v2 := paseto.NewV2()
	err = v2.Verify(token, db.GetTokenKeys().PublicKey, &restoredSession, nil)
	require.NoError(t, err)
	require.Equal(t, restoredSession.InheritedClaims, claims)

	exists, err := db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", user.ID).
		Where("token_type = ?", model.TokenTypeUserSession).
		Exists(context.TODO())
	require.True(t, exists)
	require.NoError(t, err)
}

func TestDeleteSessionByToken(t *testing.T) {
	userID, _, token, err := addTestSession()
	require.NoError(t, err)

	err = DeleteSessionByToken(context.TODO(), token)
	require.NoError(t, err)

	exists, err := db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", userID).
		Where("token_type = ?", model.TokenTypeUserSession).
		Exists(context.TODO())
	require.False(t, exists)
	require.NoError(t, err)
}

func TestDeleteSessionByID(t *testing.T) {
	userID, sessionID, _, err := addTestSession()
	require.NoError(t, err)

	err = DeleteSessionByID(context.TODO(), sessionID)
	require.NoError(t, err)

	exists, err := db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", userID).
		Where("token_type = ?", model.TokenTypeUserSession).
		Exists(context.TODO())
	require.False(t, exists)
	require.NoError(t, err)
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
	ctx := context.Background()

	expectedUser, err := addTestUser(nil)
	require.NoError(t, err)

	list, err := List(ctx)
	require.NoError(t, err)

	var actualUser model.FullUser
	for _, u := range list {
		if u.ID == expectedUser.ID {
			actualUser = u
		}
	}
	if actualUser.ID == 0 {
		require.Fail(t, "did not find expected user in list")
	}

	require.Equal(t, expectedUser.ID, actualUser.ID)
	require.Equal(t, expectedUser.DisplayName, actualUser.DisplayName)
	require.Equal(t, expectedUser.Username, actualUser.Username)
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

			aug, err := GetAgentUserGroup(context.TODO(), user.ID, workspaceID)
			require.NoError(t, err)
			if tt.workspaceAug == nil {
				nilAug := model.AgentUserGroup{
					BaseModel: schema.BaseModel{},
					ID:        0, UserID: 0,
					User: "root", UID: 0,
					Group: "root", GID: 0,
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
		Where("user_id = ?", user.ID).
		Where("token_type = ?", model.TokenTypeUserSession).
		Scan(context.TODO(), &session); err != nil {
		return 0, 0, "", fmt.Errorf("couldn't create new session: %w", err)
	}
	return user.ID, session.ID, token, nil
}

func addTestUser(aug *model.AgentUserGroup, opts ...func(*model.User)) (*model.User, error) {
	user := model.User{Username: uuid.NewString()}
	for _, opt := range opts {
		opt(&user)
	}

	uid, err := Add(context.TODO(), &user, aug)
	if err != nil {
		return nil, fmt.Errorf("couldn't create new user: %w", err)
	}
	err = db.Bun().NewSelect().Table("users").Where("id = ?", uid).Scan(context.TODO(), &user)
	return &user, err
}

func TestSetActive(t *testing.T) {
	var testUsers []model.UserID
	for _, status := range []bool{true, false} {
		u, err := addTestUser(nil, func(u *model.User) { u.Active = status })
		require.NoError(t, err)
		testUsers = append(testUsers, u.ID)
	}

	type args struct {
		updateIDs []model.UserID
		status    bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "empty ok",
			args: args{
				updateIDs: []model.UserID{},
				status:    false,
			},
		},
		{
			name: "set assorted to active",
			args: args{
				updateIDs: testUsers,
				status:    true,
			},
		},
		{
			name: "set back to inactive assorted to active",
			args: args{
				updateIDs: testUsers,
				status:    false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetActive(context.Background(), tt.args.updateIDs, tt.args.status)
			require.NoError(t, err)

			if len(tt.args.updateIDs) == 0 {
				return
			}

			for _, id := range testUsers {
				fu, err := ByID(context.Background(), id)
				require.NoError(t, err)
				require.Equal(t, fu.Active, tt.args.status)
			}
		})
	}
}

func TestProfileImage(t *testing.T) {
	u, err := addTestUser(nil)
	require.NoError(t, err)

	var fakeImage [16]byte
	_, err = rand.Read(fakeImage[:])
	require.NoError(t, err)

	_, err = db.Bun().NewInsert().Model(&UserProfileImage{
		UserID:   u.ID,
		FileData: fakeImage[:],
	}).Exec(context.Background())
	require.NoError(t, err)

	resImage, err := ProfileImage(context.Background(), u.Username)
	require.NoError(t, err)
	require.Equal(t, fakeImage[:], resImage, "received image wasn't correct")
}

func TestUpdateUserSettings(t *testing.T) {
	u, err := addTestUser(nil)
	require.NoError(t, err)

	// noop reset is fine
	err = ResetUserSetting(context.Background(), u.ID)
	require.NoError(t, err)

	// adding a few settings works
	in := []*model.UserWebSetting{
		{
			UserID:      u.ID,
			Key:         "a",
			Value:       "{}",
			StoragePath: "c",
		},
		{
			UserID:      u.ID,
			Key:         "d",
			Value:       "{\"great_setting\": \"ok\"}",
			StoragePath: "f",
		},
	}
	err = UpdateUserSetting(context.Background(), in)
	require.NoError(t, err)

	out, err := GetUserSetting(context.Background(), u.ID)
	require.NoError(t, err)
	require.Equal(t, in, out)

	// and turning one off works, too
	update := in[0]
	update.Value = ""
	err = UpdateUserSetting(context.Background(), []*model.UserWebSetting{update})
	require.NoError(t, err)

	out, err = GetUserSetting(context.Background(), u.ID)
	require.NoError(t, err)
	expected := in[1:]
	require.Equal(t, expected, out, "removing just one setting didn't work")

	// resetting them and readding them works fine
	err = ResetUserSetting(context.Background(), u.ID)
	require.NoError(t, err)

	err = UpdateUserSetting(context.Background(), expected)
	require.NoError(t, err)

	out, err = GetUserSetting(context.Background(), u.ID)
	require.NoError(t, err)
	require.Equal(t, expected, out, "removing just one setting didn't work")

	// deleting all manually works
	update = in[1]
	update.Value = ""
	err = UpdateUserSetting(context.Background(), []*model.UserWebSetting{update})
	require.NoError(t, err)

	out, err = GetUserSetting(context.Background(), u.ID)
	require.NoError(t, err)
	require.Empty(t, out, "found user web settings when all should be deleted")
}

// TestRevokeAndCreateLongLivedToken tests revoking and creating token with default lifespan.
func TestRevokeAndCreateLongLivedToken(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a LongLivedToken.
	token, err := RevokeAndCreateLongLivedToken(context.TODO(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, token)

	restoredToken := restoreTokenInfo(token, t)

	expLifespan := DefaultTokenLifespan
	actLifespan := restoredToken.Expiry.Sub(restoredToken.CreatedAt)
	require.Equal(t, expLifespan, actLifespan)

	tokenInfo, err := GetAccessToken(context.TODO(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, tokenInfo)

	// For test cleanup
	err = DeleteSessionByID(context.TODO(), tokenInfo.ID)
	require.NoError(t, err)
}

// TestRevokeAndCreateLongLivedTokenHasExpiry tests revoking and creating token with
// given lifespan and description.
func TestRevokeAndCreateLongLivedTokenHasExpiry(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a LongLivedToken with custom (Now() + 3 Months) Expiry Time.
	expLifespan := DefaultTokenLifespan * 3
	token, err := RevokeAndCreateLongLivedToken(context.TODO(), user.ID,
		WithTokenExpiry(&expLifespan), WithTokenDescription(desc))
	require.NoError(t, err)
	require.NotNil(t, token)

	restoredToken := restoreTokenInfo(token, t)

	actLifespan := restoredToken.Expiry.Sub(restoredToken.CreatedAt)
	require.Equal(t, expLifespan, actLifespan)
	require.Equal(t, desc, restoredToken.Description.String)

	tokenInfo, err := GetAccessToken(context.TODO(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, tokenInfo)

	// Delete from DB by UserID for cleanup
	err = DeleteSessionByID(context.TODO(), tokenInfo.ID)
	require.NoError(t, err)
}

// TestUpdateAccessToken tests the description and revocation status of the access token.
func TestUpdateAccessToken(t *testing.T) {
	userID, _, _, err := addTestSession()
	require.NoError(t, err)

	token, err := RevokeAndCreateLongLivedToken(context.TODO(), userID)
	require.NoError(t, err)
	require.NotNil(t, token)

	longLivedToken := restoreTokenInfo(token, t)

	// Test before updating Access token
	description := "description"
	require.False(t, longLivedToken.Revoked)
	require.NotEqual(t, description, longLivedToken.Description)

	opt := AccessTokenUpdateOptions{Description: &description, SetRevoked: true}
	tokenInfo, err := UpdateAccessToken(context.TODO(), model.TokenID(longLivedToken.ID), opt)
	require.NoError(t, err)

	// Test after updating access token
	require.True(t, tokenInfo.Revoked)
	require.Contains(t, description, tokenInfo.Description.String)

	// Delete from DB by UserID for cleanup
	err = DeleteSessionByID(context.TODO(), tokenInfo.ID)
	require.NoError(t, err)
}

// TestGetLongLivenTokenInfoByUserID tests getting long lived token info for given userId.
func TestGetAccessToken(t *testing.T) {
	user, err := addTestUser(nil)
	require.NoError(t, err)

	// Add a LongLivedToken.
	token, err := RevokeAndCreateLongLivedToken(context.TODO(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, token)

	tokenInfo, err := GetAccessToken(context.TODO(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, tokenInfo.UserID)

	restoredTokeninfo := restoreTokenInfo(token, t)
	require.Equal(t, restoredTokeninfo.ID, tokenInfo.ID)

	// Delete from DB by UserID for cleanup
	err = DeleteSessionByID(context.TODO(), tokenInfo.ID)
	require.NoError(t, err)
}

func restoreTokenInfo(token string, t *testing.T) model.UserSession {
	var restoredToken model.UserSession
	v2 := paseto.NewV2()
	err := v2.Verify(token, db.GetTokenKeys().PublicKey, &restoredToken, nil)
	require.NoError(t, err)

	return restoredToken
}
