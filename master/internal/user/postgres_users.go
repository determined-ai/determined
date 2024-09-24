package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/saas"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	// SessionDuration is how long a newly created session is valid.
	SessionDuration = 7 * 24 * time.Hour
	// PersonalGroupPostfix is the system postfix appended to the username of all personal groups.
	PersonalGroupPostfix = "DeterminedPersonalGroup"

	// DefaultTokenLifespan is how long a newly created access token is valid.
	DefaultTokenLifespan = 30 * 24 * time.Hour
)

// CurrentTimeNowInUTC stores the current time in UTC for time insertions.
var CurrentTimeNowInUTC time.Time

// ErrRemoteUserTokenExpired notifies that the remote user's token has expired.
var ErrRemoteUserTokenExpired = status.Error(codes.Unauthenticated, "remote user token expired")

// ErrAccessTokenRevoked notifies that the user's access token has been revoked.
var ErrAccessTokenRevoked = status.Error(codes.Unauthenticated, "user access token revoked")

// UserSessionOption is the return type for WithInheritedClaims helper function.
type UserSessionOption func(f *model.UserSession)

// WithInheritedClaims function will add the specified inherited claims to the user session.
func WithInheritedClaims(claims map[string]string) UserSessionOption {
	return func(s *model.UserSession) {
		s.InheritedClaims = claims
	}
}

// StartSession creates a row in the user_sessions table.
func StartSession(ctx context.Context, user *model.User, opts ...UserSessionOption) (string, error) {
	CurrentTimeNowInUTC = time.Now().UTC()

	userSession := &model.UserSession{
		UserID:    user.ID,
		Expiry:    CurrentTimeNowInUTC.Add(SessionDuration),
		CreatedAt: CurrentTimeNowInUTC,
		TokenType: model.TokenTypeUserSession,
		Revoked:   false,
	}

	for _, opt := range opts {
		opt(userSession)
	}

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(userSession).
			Column("user_id", "expiry", "created_at", "token_type", "revoked").
			Returning("id").
			Exec(ctx, &userSession.ID)
		if err != nil {
			return err
		}

		_, err = tx.NewUpdate().
			Table("users").
			SetColumn("last_auth_at", "NOW()").
			Where("id = (?)", user.ID).
			Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	})
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
			uID, err := AddUserTx(ctx, tx, user)
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
				return fmt.Errorf("error setting active status of %q: %s", updated.Username, err)
			}
		}

		if slices.Contains(toUpdate, "password_hash") {
			if _, err := tx.NewDelete().
				Table("user_sessions").
				Where("user_id = ?", updated.ID).
				Where("token_type = ?", model.TokenTypeUserSession).Exec(ctx); err != nil {
				return fmt.Errorf("error deleting user sessions: %s", err)
			}
		}

		if ug != nil {
			if err := deleteAgentUserGroup(ctx, tx, updated.ID, ug); err != nil {
				return err
			}
			if err := addAgentUserGroup(ctx, tx, updated.ID, ug); err != nil {
				return err
			}
		}

		return nil
	})
}

// SetActive changes multiple users' activation status.
func SetActive(
	ctx context.Context,
	updateIDs []model.UserID,
	activate bool,
) error {
	if len(updateIDs) > 0 {
		if _, err := db.Bun().NewUpdate().
			Table("users").
			Set("active = ?", activate).
			Where("id IN (?)", bun.In(updateIDs)).Exec(ctx); err != nil {
			return fmt.Errorf("error updating %q: %s", updateIDs, err)
		}
	}
	return nil
}

// DeleteSessionByToken deletes user session if found
// (externally managed sessions are not stored in the DB and will not be found).
func DeleteSessionByToken(ctx context.Context, token string) error {
	v2 := paseto.NewV2()
	var session model.UserSession
	// verification will fail when using external token (Jwt instead of Paseto)
	if err := v2.Verify(token, db.GetTokenKeys().PublicKey, &session, nil); err != nil {
		return nil //nolint: nilerr
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

// AddUserTx & addAgentUserGroup are helper methods for Add & Update.
// AddUserTx UPSERT's the existence of a new user.
func AddUserTx(ctx context.Context, idb bun.IDB, user *model.User) (model.UserID, error) {
	if _, err := idb.NewInsert().Model(user).ExcludeColumn("id").Returning("id").Exec(ctx); err != nil {
		if pgerr, ok := errors.Cause(err).(*pgconn.PgError); ok {
			if pgerr.Code == db.CodeUniqueViolation {
				return 0, db.ErrDuplicateRecord
			}
		}
		return 0, fmt.Errorf("error inserting user: %s", err)
	}

	personalGroup := model.Group{
		Name:    fmt.Sprintf("%d%s", user.ID, PersonalGroupPostfix),
		OwnerID: user.ID,
	}
	if _, err := idb.NewInsert().Model(&personalGroup).Exec(ctx); err != nil {
		return 0, fmt.Errorf("error inserting personal group: %s", err)
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
	if err != nil {
		return err
	}

	_, err = idb.NewUpdate().Table("users").
		Set("modified_at = NOW()").
		Where("id = ?", userID).
		Exec(ctx)
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

// A UserProfileImage row just contains the profile image data. It is probably split into another table to avoid
// medium sized images missing TOAST and slowing scans down, but I'm not sure since I didn't write this code.
type UserProfileImage struct {
	bun.BaseModel `bun:"table:user_profile_images"`
	ID            int          `bun:"id,pk,autoincrement"`
	UserID        model.UserID `bun:"user_id"`
	FileData      []byte       `bun:"file_data"`
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
		Join("LEFT JOIN user_profile_images AS img ON u.id = img.user_id").
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
func GetUserSetting(ctx context.Context, userID model.UserID) ([]*model.UserWebSetting, error) {
	var setting []*model.UserWebSetting
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
		Column("u.id", "u.display_name", "u.username", "u.admin", "u.active", "u.modified_at", "u.last_auth_at").
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
			"u.modified_at", "u.last_auth_at").
		ColumnExpr(`h.uid AS agent_uid, h.gid AS agent_gid,
		h.user_ AS agent_user, h.group_ AS agent_group`).
		Join("LEFT OUTER JOIN agent_user_groups h ON u.id = h.user_id").
		Where("u.id = ?", userID).Scan(ctx, &fu)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, err
	}

	return &fu, nil
}

// ByToken returns a user session given an authentication token. If a session belonging to a remote (SSO) user
// is found but has expired, ErrRemoteUserTokenExpired will be returned.
func ByToken(ctx context.Context, token string, ext *model.ExternalSessions) (
	*model.User, *model.UserSession, error,
) {
	var session model.UserSession

	if ext.JwtKey != "" {
		return saas.GetAndMaybeProvisionUserByToken(ctx, token, ext)
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

	if session.Expiry.Before(time.Now().UTC()) {
		var isRemote bool
		if err := db.Bun().NewSelect().
			Model(&model.User{}).
			Column("remote").
			Where("id = ?", session.UserID).
			Scan(ctx, &isRemote); err != nil {
			return nil, nil, db.ErrNotFound
		}

		// flag remote users as expired, so they are redirected to the SSO login page
		// instead of returning an error
		if isRemote {
			return nil, nil, ErrRemoteUserTokenExpired
		}

		return nil, nil, db.ErrNotFound
	}

	if session.TokenType == model.TokenTypeAccessToken && session.Revoked {
		return nil, nil, ErrAccessTokenRevoked
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

// BySessionID looks up a user by session ID in the database.
func BySessionID(ctx context.Context, sessionID model.SessionID) (*model.User, error) {
	var user model.User
	err := db.Bun().NewSelect().
		Table("users").
		ColumnExpr("users.*").
		Join("JOIN user_sessions ON user_sessions.user_id = users.id").
		Where("user_sessions.id = ?", sessionID).Scan(ctx, &user)
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching session (%d)", sessionID)
	}
	return &user, nil
}

// AccessTokenOption is the return type for WithTokenExpiry helper function.
// It takes a pointer to model.UserSession and modifies it.
// It’s used to apply optional settings to the AccessToken object.
type AccessTokenOption func(f *model.UserSession)

// WithTokenExpiry function will add specified expiresAt (if any) to the access token table.
func WithTokenExpiry(expiry *time.Duration) AccessTokenOption {
	return func(s *model.UserSession) {
		s.Expiry = CurrentTimeNowInUTC.Add(*expiry)
	}
}

// WithTokenDescription function will add specified description (if any) to the access token table.
func WithTokenDescription(description string) AccessTokenOption {
	return func(s *model.UserSession) {
		if description == "" {
			return
		}
		s.Description = null.StringFrom(description)
	}
}

// RevokeAndCreateAccessToken creates/overwrites a access token and store in
// user_sessions db.
func RevokeAndCreateAccessToken(
	ctx context.Context, userID model.UserID, opts ...AccessTokenOption,
) (string, error) {
	CurrentTimeNowInUTC = time.Now().UTC()
	// Populate the default values in the model.
	accessToken := &model.UserSession{
		UserID:      userID,
		CreatedAt:   CurrentTimeNowInUTC,
		Expiry:      CurrentTimeNowInUTC.Add(DefaultTokenLifespan),
		TokenType:   model.TokenTypeAccessToken,
		Description: null.StringFromPtr(nil),
		Revoked:     false,
	}

	// Update the optional ExpiresAt field (if passed)
	for _, opt := range opts {
		opt(accessToken)
	}

	var token string

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// AccessTokens should have a 1:1 relationship with users, if a user creates a new token,
		// revoke the previous token if it exists.
		_, err := tx.NewUpdate().
			Table("user_sessions").
			Set("revoked = true").
			Where("user_id = ?", userID).
			Where("token_type = ?", model.TokenTypeAccessToken).
			Exec(ctx)
		if err != nil {
			return err
		}

		// A new row is inserted into the user_sessions table, and the ID of the
		// inserted row is returned and stored in user_sessions.ID.
		_, err = tx.NewInsert().
			Model(accessToken).
			Column("user_id", "expiry", "created_at", "token_type", "revoked", "description").
			Returning("id").
			Exec(ctx, &accessToken.ID)
		if err != nil {
			return err
		}

		// A Paseto token is generated using the accessToken object and the private key.
		v2 := paseto.NewV2()
		privateKey := db.GetTokenKeys().PrivateKey
		token, err = v2.Sign(privateKey, accessToken, nil)
		if err != nil {
			return fmt.Errorf("failed to generate user authentication token: %s", err)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

// AccessTokenUpdateOptions is the set of mutable fields for an Access Token record.
type AccessTokenUpdateOptions struct {
	Description *string
	SetRevoked  bool
}

// UpdateAccessToken updates the description and revocation status of the access token.
func UpdateAccessToken(
	ctx context.Context, tokenID model.TokenID, options AccessTokenUpdateOptions,
) (*model.UserSession, error) {
	var tokenInfo model.UserSession
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewSelect().Table("user_sessions").
			Where("id = ?", tokenID).Where("token_type = ?", model.TokenTypeAccessToken).
			Scan(ctx, &tokenInfo)
		if err != nil {
			return err
		}

		if tokenInfo.Revoked {
			return fmt.Errorf("unable to update revoked token with ID %v", tokenID)
		}

		if options.Description != nil {
			tokenInfo.Description = null.StringFrom(*options.Description)
		}

		if options.SetRevoked {
			tokenInfo.Revoked = true
		}

		_, err = tx.NewUpdate().
			Model(&tokenInfo).
			Column("description", "revoked").
			Where("id = ?", tokenID).
			Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &tokenInfo, nil
}

// GetAccessToken returns the active token info from the table with the user_id.
func GetAccessToken(ctx context.Context, userID model.UserID) (
	*model.UserSession, error,
) {
	var tokenInfo model.UserSession // To store the token info for the given user_id

	// Execute the query to fetch the active token info for the given user_id
	switch err := db.Bun().NewSelect().Table("user_sessions").
		Where("user_id = ?", userID).Where("revoked = ?", false).
		Where("token_type = ?", model.TokenTypeAccessToken).
		Scan(ctx, &tokenInfo); {
	case err != nil:
		return nil, err
	default:
		return &tokenInfo, nil
	}
}
