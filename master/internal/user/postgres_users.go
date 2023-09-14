package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

// SessionDuration is how long a newly created session is valid.
const SessionDuration = 7 * 24 * time.Hour

// PersonalGroupPostfix is the system postfix appended to the username of all personal groups.
const PersonalGroupPostfix = "DeterminedPersonalGroup"

// StartSession creates a row in the user_sessions table.
func StartSession(ctx context.Context, user *model.User) (string, error) {
	userSession := &model.UserSession{
		UserID: user.ID,
		Expiry: time.Now().Add(SessionDuration),
	}

	_, err := db.Bun().NewInsert().
		Model(userSession).
		Column("user_id", "expiry").
		Returning("id").
		Exec(ctx, &userSession.ID)
	if err != nil {
		return "", err
	}

	v2 := paseto.NewV2()
	privateKey := db.GetTokenKeys().PrivateKey
	token, err := v2.Sign(privateKey, userSession, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate user authentication token: %s", err)
	}
	return token, nil
}

// Add creates a new user, adding it to the User & AgentUserGroup tables.
func Add(
	ctx context.Context,
	user *model.User,
	ug *model.AgentUserGroup,
) (model.UserID, error) {
	var userID model.UserID
	return userID, db.Bun().RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			uID, err := addUser(ctx, tx, user)
			if err != nil {
				return err
			}
			userID = uID
			return addAgentUserGroup(ctx, tx, userID, ug)
		})
}

// Update updates an existing user.  `toUpdate` names the fields to update.
func Update(
	ctx context.Context,
	updated *model.User,
	toUpdate []string,
	ug *model.AgentUserGroup,
) error {
	return db.Bun().RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if len(toUpdate) > 0 {
			if _, err := tx.NewUpdate().
				Model(updated).
				Column(toUpdate...).
				Where("id = ?", updated.ID).Exec(ctx); err != nil {
				return fmt.Errorf("error updating %q: %s", updated.Username, err)
			}
		}

		if slices.Contains(toUpdate, "password_hash") {
			if _, err := tx.NewDelete().
				Table("user_sessions").
				Where("user_id = ?", updated.ID).Exec(ctx); err != nil {
				return fmt.Errorf("error deleting user sessions: %s", err)
			}
		}

		if err := deleteAgentUserGroup(ctx, tx, updated.ID, ug); err != nil {
			return err
		}

		if err := addAgentUserGroup(ctx, tx, updated.ID, ug); err != nil {
			return err
		}

		return nil
	})
}

// DeleteSessionByToken deletes user session if found
// (externally managed sessions are not stored in the DB and will not be found).
func DeleteSessionByToken(ctx context.Context, token string) error {
	v2 := paseto.NewV2()
	var session model.UserSession
	// verification will fail when using external token (Jwt instead of Paseto)
	if err := v2.Verify(token, db.GetTokenKeys().PublicKey, &session, nil); err != nil {
		return nil
	}
	return DeleteSessionByID(ctx, session.ID)
}

// DeleteSessionByID deletes the user session with the given ID.
func DeleteSessionByID(ctx context.Context, sessionID model.SessionID) error {
	_, err := db.Bun().NewDelete().
		Table("user_sessions").
		Where("id = ?", sessionID).
		Exec(ctx)
	return err
}

// addUser & addAgentUserGroup are helper methods for Add & Update.
// addUser UPSERT's the existence of a new user.
func addUser(ctx context.Context, idb bun.IDB, user *model.User) (model.UserID, error) {
	if _, err := idb.NewInsert().Model(user).Returning("id").Exec(ctx); err != nil {
		return 0, fmt.Errorf("error inserting user: %s", err)
	}

	personalGroup := model.Group{
		Name:    fmt.Sprintf("%d%s", user.ID, PersonalGroupPostfix),
		OwnerID: user.ID,
	}
	if _, err := idb.NewInsert().Model(&personalGroup).Exec(ctx); err != nil {
		return 0, fmt.Errorf("error inserting personal grou: %s", err)
	}

	groupMembership := model.GroupMembership{
		UserID:  user.ID,
		GroupID: personalGroup.ID,
	}
	if _, err := idb.NewInsert().Model(&groupMembership).Exec(ctx); err != nil {
		return 0, fmt.Errorf("error adding user to personal group: %s", err)
	}
	return user.ID, nil
}

// addAgentUserGroup UPSERT's the existence of an agent user group.
func addAgentUserGroup(
	ctx context.Context,
	idb bun.IDB,
	userID model.UserID,
	ug *model.AgentUserGroup,
) error {
	if ug == nil || ug == (&model.AgentUserGroup{}) {
		return nil
	}

	next := *ug
	next.UserID = userID

	exists, _ := idb.NewSelect().
		Table("agent_user_groups").
		Where("user_id = ?", userID).
		Exists(ctx)
	if exists {
		_, err := idb.NewUpdate().Model(&next).
			Returning("id").Where("user_id = ?", userID).Exec(ctx)
		return err
	}
	_, err := idb.NewInsert().Model(&next).Returning("id").Exec(ctx)
	return err
}

// deleteAgentUserGroup is a helper method for Update.
func deleteAgentUserGroup(
	ctx context.Context,
	idb bun.IDB,
	userID model.UserID,
	ug *model.AgentUserGroup,
) error {
	if ug == nil {
		return nil
	}
	_, err := idb.NewDelete().
		Table("agent_user_groups").
		Where("user_id = ?", userID).Exec(ctx)
	return err
}

// ProfileImage returns the profile picture associated with the user.
func ProfileImage(ctx context.Context, username string) (photo []byte, err error) {
	type photoRow struct {
		Photo []byte
	}
	var userPhoto photoRow
	err = db.Bun().NewSelect().
		TableExpr("users AS u").
		ColumnExpr("file_data AS photo").
		Join("LEFT JOIN user_profile_images AS img ON u.id = img.id").
		Where("u.username = ?", username).
		Limit(1).Scan(ctx, &userPhoto)
	return userPhoto.Photo, err
}

// UpdateUsername updates an existing user's username.
func UpdateUsername(ctx context.Context, userID *model.UserID, newUsername string) error {
	_, err := db.Bun().NewUpdate().
		Model(&model.User{}).
		Set("username = ?", newUsername).
		Where("id = ?", userID).
		Exec(ctx)
	return err
}

// UpdateUserSetting updates user setting.
func UpdateUserSetting(ctx context.Context, settings []*model.UserWebSetting) error {
	return db.Bun().RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var err error
		for _, setting := range settings {
			if len(setting.Value) == 0 {
				_, err = tx.NewDelete().
					Model(setting).
					Where("user_id = ?", setting.UserID).
					Where("storage_path = ?", setting.StoragePath).
					Where("key = ?", setting.Key).Exec(ctx)
			} else {
				_, err = tx.NewInsert().
					Model(setting).
					On("CONFLICT (user_id, key, storage_path) DO UPDATE").
					Set("value = EXCLUDED.value").Exec(ctx)
			}
		}
		return err
	})
}

// GetUserSetting gets user setting.
func GetUserSetting(ctx context.Context, userID model.UserID) ([]*userv1.UserWebSetting, error) {
	var setting []*userv1.UserWebSetting
	err := db.Bun().NewSelect().
		Model(&setting).
		Where("user_id = ?", userID).
		Scan(ctx)
	return setting, err
}

// ResetUserSetting resets user setting.
func ResetUserSetting(ctx context.Context, userID model.UserID) error {
	_, err := db.Bun().NewDelete().
		Model(&model.UserWebSetting{}).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

// List returns all of the users in the database.
func List(ctx context.Context) (values []model.FullUser, err error) {
	err = db.Bun().NewSelect().TableExpr("users AS u").
		Column("u.id", "u.display_name", "u.username", "u.admin", "u.active", "u.modified_at").
		ColumnExpr(`h.uid AS agent_uid, h.gid AS agent_gid, 
		h.user_ AS agent_user, h.group_ AS agent_group`).
		Join("LEFT OUTER JOIN agent_user_groups h ON u.id = h.user_id").
		Scan(ctx, &values)
	return values, err
}

// ByID returns the full user for a given ID.
func ByID(ctx context.Context, userID model.UserID) (*model.FullUser, error) {
	var fu model.FullUser
	err := db.Bun().NewSelect().TableExpr("users AS u").
		Column("u.id", "u.username",
			"u.display_name", "u.admin",
			"u.active", "u.remote",
			"u.modified_at").
		ColumnExpr(`h.uid AS agent_uid, h.gid AS agent_gid, 
		h.user_ AS agent_user, h.group_ AS agent_group`).
		Join("LEFT OUTER JOIN agent_user_groups h ON u.id = h.user_id").
		Where("u.id = ?", userID).Scan(ctx, &fu)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, db.ErrNotFound
		}
		return nil, err
	}

	return &fu, nil
}

// ByToken returns a user session given an authentication token.
func ByToken(ctx context.Context, token string, ext *model.ExternalSessions) (
	*model.User, *model.UserSession, error,
) {
	var session model.UserSession

	if ext.JwtKey != "" {
		return UserByExternalToken(token, ext)
	}

	v2 := paseto.NewV2()
	if err := v2.Verify(token, db.GetTokenKeys().PublicKey, &session, nil); err != nil {
		return nil, nil, db.ErrNotFound
	}

	if err := db.Bun().NewSelect().
		Model(&session).
		Where("id = ?", session.ID).
		Scan(ctx); err != nil {
		return nil, nil, err
	}

	if session.Expiry.Before(time.Now()) {
		return nil, nil, db.ErrNotFound
	}

	var user model.User
	err := db.Bun().NewSelect().
		Table("users").
		ColumnExpr("users.*").
		Join("JOIN user_sessions ON user_sessions.user_id = users.id").
		Where("user_sessions.id = ?", session.ID).Scan(ctx, &user)
	if err != nil {
		return nil, nil, err
	}

	return &user, &session, nil
}

// ByUsername looks up a user by name in the database.
func ByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	switch err := db.Bun().NewSelect().
		Model(&user).
		Where("username = ?", username).
		Scan(ctx); {
	case errors.Is(err, sql.ErrNoRows):
		return nil, db.ErrNotFound
	case err != nil:
		return nil, err
	default:
		return &user, nil
	}
}
