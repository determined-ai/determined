//go:build integration
// +build integration

package user

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// Tests for postgres_scim_users.go.
func TestAddSCIMUser(t *testing.T) {
	testUUID := model.NewUUID().String()

	ctx := context.Background()
	cases := []struct {
		name      string
		users     []*model.SCIMUser
		errString string
	}{
		{"simple-case", []*model.SCIMUser{mockSCIMUser(t)}, ""},
		{"multiples-case", []*model.SCIMUser{{
			Username:   model.NewUUID().String(),
			ExternalID: "multiples-external-id",
			Name:       model.SCIMName{GivenName: "John", FamilyName: "Multiple"},
			Emails: []model.SCIMEmail{
				{Type: "personal", SValue: "value-1", Primary: true},
				{Type: "personal", SValue: "value-2", Primary: false},
				{Type: "personal", SValue: "value-3", Primary: false},
			},
			Active:       true,
			PasswordHash: null.StringFrom("password"),
			RawAttributes: map[string]interface{}{
				"attribute1": true,
				"attribute2": "false",
				"attribute3": []interface{}{"a", "b", "c"},
			},
		}}, ""},
		{"duplicate-case", []*model.SCIMUser{
			mockSCIMUserWithUsername(t, testUUID),
			mockSCIMUserWithUsername(t, testUUID),
		}, db.ErrDuplicateRecord.Error()},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for _, v := range tt.users {
				addedUser, err := AddSCIMUser(ctx, v)
				if tt.errString != "" && err != nil {
					require.Contains(t, err.Error(), tt.errString)
					continue
				}

				require.NoError(t, err)
				dbUser, err := SCIMUserByID(ctx, db.Bun(), v.ID)
				require.NoError(t, err)
				matchUsers(t, addedUser, dbUser)

				// make sure the user table is updated too
				var u *model.FullUser
				u, err = ByID(ctx, addedUser.UserID)
				require.NoError(t, err)
				require.Equal(t, dbUser.Active, u.Active)
				require.Equal(t, dbUser.Username, u.Username)
				require.Equal(t, dbUser.PasswordHash, u.ToUser().PasswordHash)
				require.Equal(t, dbUser.DisplayName, u.DisplayName)
			}
		})
	}
}

func TestSCIMUserList(t *testing.T) {
	uuid1 := model.NewUUID().String()
	uuid2 := model.NewUUID().String()
	uuid3 := model.NewUUID().String()

	ctx := context.Background()
	cases := []struct {
		name            string
		usernameToMatch string
		usernames       []string
		count           int
		startIndex      int
	}{
		{"simple-case", "", []string{}, 0, 1},
		{"one-user-added", uuid1, []string{uuid1}, 1, 1},
		{"two-diff-users-added", uuid2, []string{uuid2, model.NewUUID().String()}, 1, 1},
		{"two-diff-users-returned", "", []string{
			model.NewUUID().String(),
			model.NewUUID().String(), model.NewUUID().String(),
		}, 1, 2},
		{"out-of-bounds-index", uuid3, []string{uuid3, model.NewUUID().String()}, 2, 2},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			expectedUsers := []*model.SCIMUser{}
			for idx, u := range tt.usernames {
				addedUser, err := AddSCIMUser(ctx, mockSCIMUserWithUsername(t, u))
				require.NoError(t, err)
				if idx+1 >= tt.startIndex && idx < tt.count {
					expectedUsers = append(expectedUsers, addedUser)
				}
			}

			actualUsers, err := SCIMUserList(ctx, tt.startIndex, tt.count, tt.usernameToMatch)
			require.NoError(t, err)
			require.Equal(t, tt.startIndex, actualUsers.StartIndex)
			if tt.name == "out-of-bounds-index" {
				require.Empty(t, actualUsers.Resources)
			} else {
				require.Subset(t, usernameList(actualUsers.Resources), usernameList(expectedUsers))
			}
		})
	}
}

func TestSCIMUserByID(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		user      *model.SCIMUser
		errString string
	}{
		{"simple-case", mockSCIMUser(t), ""},
		{"error-not-found", nil, "not found"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			id := model.UUID{}
			if tt.user != nil {
				addedUser, err := AddSCIMUser(ctx, tt.user)
				require.NoError(t, err)
				id = addedUser.ID
			}
			scimUser, err := SCIMUserByID(ctx, db.Bun(), id)
			if tt.errString != "" {
				require.Nil(t, scimUser)
				require.ErrorContains(t, err, tt.errString)
			} else {
				require.NotNil(t, scimUser)
				require.NoError(t, err)
			}
		})
	}
}

func TestUserByAttribute(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		errString string
	}{
		{"simple-case", ""},
		{"error-not-found", "not found"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			val := model.NewUUID()
			user := mockSCIMUser(t)
			user.RawAttributes = map[string]interface{}{"id": val}

			addedUser, err := AddSCIMUser(ctx, user)
			require.NoError(t, err)

			if tt.errString != "" {
				_, err := scimUserByAttribute(ctx, "user_id", "bogus-value")
				require.Contains(t, err.Error(), tt.errString)

				_, err = UserBySCIMAttribute(ctx, "user_id", "bogus-value")
				require.Contains(t, err.Error(), tt.errString)
			} else {
				// test scimUserByAttribute
				scimUser, err := scimUserByAttribute(ctx, "id", fmt.Sprint(val))
				require.NoError(t, err)
				require.Equal(t, addedUser.Username, scimUser.Username)

				// test userBySCIMAttribute
				u, err := UserBySCIMAttribute(ctx, "id", fmt.Sprint(val))
				require.NoError(t, err)
				require.Equal(t, addedUser.UserID, u.ID)
			}
		})
	}
}

func TestSetSCIMUser(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name        string
		updatedUser *model.SCIMUser
		errString   string
		matchUUID   bool
	}{
		{"simple-case", mockSCIMUser(t), "", true},
		{"simple-case", mockSCIMUser(t), "does not match updated user ID", false},
		{"empty-set", &model.SCIMUser{}, "duplicate key value", true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			addedUser, err := AddSCIMUser(ctx, mockSCIMUser(t))
			require.NoError(t, err)

			if tt.matchUUID {
				tt.updatedUser.ID = addedUser.ID
			}

			user, err := SetSCIMUser(ctx, addedUser.ID.String(), tt.updatedUser)
			if err != nil {
				require.Contains(t, err.Error(), tt.errString)
			} else {
				require.NoError(t, err)
				matchUsers(t, tt.updatedUser, user)
				require.Equal(t, addedUser.ID, user.ID)
			}
		})
	}
}

func TestUpdateUserAndDeleteSession(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name        string
		fields      []string
		updatedUser *model.SCIMUser
		matchID     bool
		errString   string
	}{
		{"simple-case-one-field", []string{"username"}, mockSCIMUser(t), true, ""},
		{"multiple-fields", []string{"name", "emails", "username"}, mockSCIMUser(t), true, ""},
		{"id-not-found", []string{"username"}, mockSCIMUser(t), false, "does not match updated user ID"},
		{"id-not-found", []string{"display_name"}, mockSCIMUser(t), false, "does not match updated user ID"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Adding a mock test session -- to test for deletion later
			// Add a user.
			addedUser, err := AddSCIMUser(ctx, mockSCIMUser(t))
			require.NoError(t, err)

			var user model.User
			err = db.Bun().NewSelect().Table("users").Where("id = ?", addedUser.UserID).Scan(ctx, &user)
			require.NoError(t, err)

			// Add a session.
			var session model.UserSession
			_, err = StartSession(ctx, &user)
			require.NoError(t, err)

			err = db.Bun().NewSelect().Table("user_sessions").
				Where("user_id = ?", user.ID).Scan(ctx, &session)
			require.NoError(t, err)

			if tt.matchID {
				tt.updatedUser.ID = addedUser.ID
			}

			scimUser, err := UpdateUserAndDeleteSession(ctx, addedUser.ID.String(), tt.updatedUser, tt.fields)
			if tt.errString != "" {
				require.Contains(t, err.Error(), tt.errString)
			} else {
				require.NoError(t, err)
				for _, v := range tt.fields {
					switch v {
					case "username":
						require.Equal(t, tt.updatedUser.Username, scimUser.Username)
					case "emails":
						require.Equal(t, tt.updatedUser.Emails, scimUser.Emails)
					case "name":
						require.Equal(t, tt.updatedUser.Name, scimUser.Name)
					case "display_name":
						// Since display name isn't a column that exists in the SCIM User DB table,
						// check that it's updated in the user table correctly.
						u, err := ByID(ctx, scimUser.UserID)
						require.NoError(t, err)
						require.NotEqual(t, addedUser.DisplayName, u.DisplayName)
						require.Equal(t, tt.updatedUser.DisplayName, u.DisplayName)
					}
				}
			}

			_, err = db.Bun().NewSelect().Table("user_sessions").
				Where("user_id = ?", user.ID).Exec(context.Background())
			require.ErrorAs(t, errors.New("Receive unexpected error: bun: Model(nil)"), &err)
		})
	}
}

func mockSCIMUser(t *testing.T) *model.SCIMUser {
	return mockSCIMUserWithUsername(t, model.NewUUID().String())
}

func mockSCIMUserWithUsername(t *testing.T, username string) *model.SCIMUser {
	user := &model.SCIMUser{
		Username:     username,
		DisplayName:  null.StringFrom(fmt.Sprintf("disp-%s", username)),
		ExternalID:   fmt.Sprintf("external-id-%s", username),
		Name:         model.SCIMName{GivenName: "John", FamilyName: username},
		Emails:       []model.SCIMEmail{{Type: "personal", SValue: fmt.Sprintf("value-%s", username), Primary: true}},
		Active:       true,
		PasswordHash: null.StringFrom("password"),
	}

	return user
}

func usernameList(l []*model.SCIMUser) []string {
	res := []string{}
	for _, v := range l {
		res = append(res, v.Username)
	}
	return res
}

func matchUsers(t *testing.T, a *model.SCIMUser, b *model.SCIMUser) {
	// because only certain fields are written to the db
	require.Equal(t, a.Username, b.Username)
	require.Equal(t, a.ExternalID, b.ExternalID)
	require.Equal(t, a.Name, b.Name)
	require.Equal(t, a.Emails, b.Emails)
	require.Equal(t, a.Active, b.Active)
	require.Equal(t, a.RawAttributes, b.RawAttributes)
}
