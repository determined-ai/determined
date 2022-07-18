package db

import (
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

// SessionDuration is how long a newly created session is valid.
const SessionDuration = 7 * 24 * time.Hour

// StartUserSession creates a row in the user_sessions table.
func (db *PgDB) StartUserSession(user *model.User) (string, error) {
	userSession := &model.UserSession{
		UserID: user.ID,
		Expiry: time.Now().Add(SessionDuration),
	}

	query := "INSERT INTO user_sessions (user_id, expiry) VALUES (:user_id, :expiry) RETURNING id"
	if err := db.namedGet(&userSession.ID, query, *userSession); err != nil {
		return "", err
	}

	v2 := paseto.NewV2()
	token, err := v2.Sign(db.tokenKeys.PrivateKey, userSession, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate user authentication token")
	}
	return token, nil
}

// DeleteUserSessionByToken deletes user session if found
// (externally managed sessions are not stored in the DB and will not be found).
func (db *PgDB) DeleteUserSessionByToken(token string) error {
	v2 := paseto.NewV2()
	var session model.UserSession
	// verification will fail when using external token (Jwt instead of Paseto)
	if err := v2.Verify(token, db.tokenKeys.PublicKey, &session, nil); err != nil {
		return nil
	}
	return db.DeleteUserSessionByID(session.ID)
}

// UserByToken returns a user session given an authentication token.
func (db *PgDB) UserByToken(token string, ext *model.ExternalSessions) (
	*model.User, *model.UserSession, error,
) {
	if ext.JwtKey != "" {
		return db.UserByExternalToken(token, ext)
	}

	v2 := paseto.NewV2()

	var session model.UserSession
	err := v2.Verify(token, db.tokenKeys.PublicKey, &session, nil)
	if err != nil {
		return nil, nil, ErrNotFound
	}

	query := `SELECT * FROM user_sessions WHERE id=$1`
	if err := db.query(query, &session, session.ID); errors.Cause(err) == ErrNotFound {
		return nil, nil, ErrNotFound
	} else if err != nil {
		return nil, nil, err
	}

	if session.Expiry.Before(time.Now()) {
		return nil, nil, ErrNotFound
	}

	var user model.User
	if err := db.query(`
SELECT users.* FROM users
JOIN user_sessions ON user_sessions.user_id = users.id
WHERE user_sessions.id=$1`, &user, session.ID); errors.Cause(err) == ErrNotFound {
		return nil, nil, ErrNotFound
	} else if err != nil {
		return nil, nil, err
	}

	return &user, &session, nil
}

// UserByExternalToken returns a user session derived from an external authentication token.
func (db *PgDB) UserByExternalToken(tokenText string,
	ext *model.ExternalSessions,
) (*model.User, *model.UserSession, error) {
	type externalToken struct {
		*jwt.StandardClaims
		Email string
	}

	token, err := jwt.ParseWithClaims(tokenText, &externalToken{},
		func(token *jwt.Token) (interface{}, error) {
			var publicKey rsa.PublicKey
			err := json.Unmarshal([]byte(ext.JwtKey), &publicKey)
			if err != nil {
				log.Errorf("error parsing JWT key: %s", err.Error())
				return nil, err
			}
			return &publicKey, nil
		},
	)
	if err != nil {
		return nil, nil, err
	}
	claims := token.Claims.(*externalToken)

	// Access control logic can be applied here

	tx, err := db.sql.Beginx()
	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("error during rollback: %v", rErr)
		}
	}()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	user, err := db.UserByUsername(claims.Email)
	if err != nil {
		if err != ErrNotFound {
			return nil, nil, err
		}
		user = &model.User{
			Username:     claims.Email,
			PasswordHash: null.NewString("", false),
			Admin:        true,
			Active:       true,
		}
		userID, err := addUser(tx, user)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		user.ID = userID
		if err := tx.Commit(); err != nil {
			return nil, nil, errors.WithStack(err)
		}
		tx = nil
	}

	session := &model.UserSession{
		ID:     model.SessionID(user.ID),
		UserID: user.ID,
		Expiry: time.Unix(claims.ExpiresAt, 0),
	}

	return user, session, nil
}

// DeleteUserSessionByID deletes the user session with the given ID.
func (db *PgDB) DeleteUserSessionByID(sessionID model.SessionID) error {
	_, err := db.sql.Exec("DELETE FROM user_sessions WHERE id=$1", sessionID)
	return err
}

// UserByUsername looks up a user by name in the database.
func (db *PgDB) UserByUsername(username string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE username=$1`
	if err := db.query(query, &user, strings.ToLower(username)); errors.Cause(err) == ErrNotFound {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

func addUser(tx *sqlx.Tx, user *model.User) (model.UserID, error) {
	stmt, err := tx.PrepareNamed(`
INSERT INTO users
(username, admin, active, password_hash)
VALUES (:username, :admin, :active, :password_hash)
RETURNING id`)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer stmt.Close()

	if err := stmt.QueryRowx(user).Scan(&user.ID); err != nil {
		if pgerr, ok := errors.Cause(err).(*pgconn.PgError); ok {
			if pgerr.Code == uniqueViolation {
				return 0, ErrDuplicateRecord
			}
		}
		return 0, errors.Wrapf(err, "error creating user %v", err)
	}

	return user.ID, nil
}

func addAgentUserGroup(tx *sqlx.Tx, userID model.UserID, ug *model.AgentUserGroup) error {
	next := *ug
	next.UserID = userID

	stmt, err := tx.PrepareNamed(`
INSERT INTO agent_user_groups
(user_id, user_, uid, group_, gid)
VALUES (:user_id, :user_, :uid, :group_, :gid)`)
	if err != nil {
		return errors.WithStack(err)
	}
	defer stmt.Close()

	if _, err := stmt.Exec(next); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func deleteAgentUserGroup(tx *sqlx.Tx, userID model.UserID) error {
	query := "DELETE FROM agent_user_groups WHERE user_id = $1"
	if _, err := tx.Exec(query, userID); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// AddUser creates a new user.
func (db *PgDB) AddUser(user *model.User, ug *model.AgentUserGroup) (model.UserID, error) {
	tx, err := db.sql.Beginx()
	if err != nil {
		return 0, errors.WithStack(err)
	}

	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("error during rollback: %v", rErr)
		}
	}()

	userID, err := addUser(tx, user)
	if err != nil {
		return 0, err
	}

	if ug != nil {
		if err := addAgentUserGroup(tx, userID, ug); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.WithStack(err)
	}

	tx = nil
	return userID, nil
}

// UpdateUser updates an existing user.  `toUpdate` names the fields to update.
func (db *PgDB) UpdateUser(updated *model.User, toUpdate []string, ug *model.AgentUserGroup) error {
	tx, err := db.sql.Beginx()
	if err != nil {
		return errors.Wrap(err, "error starting transaction")
	}
	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("error during rollback: %v", rErr)
		}
	}()

	if len(toUpdate) > 0 {
		query := fmt.Sprintf(
			"UPDATE users %v WHERE id = :id",
			setClause(toUpdate))

		if _, err = tx.NamedExec(query, updated); err != nil {
			return errors.Wrapf(err, "updating %q", updated.Username)
		}
	}

	var updatePassword bool
	for _, e := range toUpdate {
		if e == "password_hash" {
			updatePassword = true
			break
		}
	}

	if updatePassword {
		query := "DELETE FROM user_sessions WHERE user_id = $1"
		if _, err = tx.Exec(query, updated.ID); err != nil {
			return errors.Wrap(err, "error deleting user sessions")
		}
	}

	if ug != nil {
		if err = deleteAgentUserGroup(tx, updated.ID); err != nil {
			return err
		}

		if err = addAgentUserGroup(tx, updated.ID, ug); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "error committing change to user")
	}

	tx = nil

	return nil
}

// UpdateUsername updates an existing user's username.
func (db *PgDB) UpdateUsername(userID *model.UserID, newUsername string) error {
	if _, err := db.sql.Exec(
		"UPDATE users SET username = $1 WHERE id = $2", newUsername, userID); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// UserList returns all of the users in the database.
func (db *PgDB) UserList() (values []model.FullUser, err error) {
	err = db.Query("list_users", &values)
	return values, err
}

// UserImage returns the profile picture associated with the user.
func (db *PgDB) UserImage(username string) (photo []byte, err error) {
	type photoRow struct {
		Photo []byte
	}
	var userPhoto photoRow
	err = db.Query("get_user_image", &userPhoto, username)
	return userPhoto.Photo, err
}

// UserByID returns the full user for a given ID.
func (db *PgDB) UserByID(userID model.UserID) (*model.FullUser, error) {
	var fu model.FullUser
	if err := db.query(`
SELECT
	u.id, u.username, u.display_name, u.admin, u.active,
	h.uid AS agent_uid, h.gid AS agent_gid, h.user_ AS agent_user, h.group_ AS agent_group
FROM users u
LEFT OUTER JOIN agent_user_groups h ON (u.id = h.user_id)
WHERE u.id = $1`, &fu, userID); err != nil {
		return nil, err
	}

	return &fu, nil
}

// AgentUserGroup returns the AgentUserGroup for the user or nil if none exists.
func (db *PgDB) AgentUserGroup(userID model.UserID) (*model.AgentUserGroup, error) {
	var ug model.AgentUserGroup
	if err := db.query(`
SELECT h.user_id, h.user_, h.uid, h.group_, h.gid
FROM agent_user_groups h, users u
WHERE u.id = $1 AND u.id = h.user_id`, &ug, userID); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &ug, nil
}

func (db *PgDB) initAuthKeys() error {
	switch storedKeys, err := db.AuthTokenKeypair(); {
	case err != nil:
		return errors.Wrap(err, "error retrieving auth token keypair")
	case storedKeys == nil:
		publicKey, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return errors.Wrap(err, "error creating auth token keypair")
		}
		tokenKeypair := model.AuthTokenKeypair{PublicKey: publicKey, PrivateKey: privateKey}
		err = db.AddAuthTokenKeypair(&tokenKeypair)
		if err != nil {
			return errors.Wrap(err, "error saving auth token keypair")
		}
		db.tokenKeys = &tokenKeypair
	default:
		db.tokenKeys = storedKeys
	}
	return nil
}

// AddAuthTokenKeypair adds the new auth token keypair.
func (db *PgDB) AddAuthTokenKeypair(tokenKeypair *model.AuthTokenKeypair) error {
	return db.namedExecOne(`
INSERT INTO auth_token_keypair (public_key, private_key)
VALUES (:public_key, :private_key)`, *tokenKeypair)
}

// AuthTokenKeypair gets the existing auth token keypair.
func (db *PgDB) AuthTokenKeypair() (*model.AuthTokenKeypair, error) {
	var tokenKeypair model.AuthTokenKeypair
	switch err := db.query("SELECT * FROM auth_token_keypair", &tokenKeypair); {
	case errors.Cause(err) == ErrNotFound:
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &tokenKeypair, nil
	}
}

// UpdateUserSetting updates user setting.
func UpdateUserSetting(setting *model.UserWebSetting) error {
	if len(setting.Value) == 0 {
		_, err := Bun().NewDelete().Model(setting).Where(
			"user_id = ?", setting.UserID).Where(
			"storage_path = ?", setting.StoragePath).Where(
			"key = ?", setting.Key).Exec(context.TODO())
		return err
	}

	_, err := Bun().NewInsert().Model(setting).On("CONFLICT (user_id, key, storage_path) DO UPDATE").
		Set("value = EXCLUDED.value").Exec(context.TODO())
	return err
}

// GetUserSetting gets user setting.
func GetUserSetting(userID model.UserID) ([]*userv1.UserWebSetting, error) {
	setting := []*userv1.UserWebSetting{}
	err := Bun().NewSelect().Model(&setting).Where("user_id = ?", userID).Scan(context.TODO())
	return setting, err
}

// ResetUserSetting resets user setting.
func ResetUserSetting(userID model.UserID) error {
	var setting model.UserWebSetting
	_, err := Bun().NewDelete().Model(&setting).Where("user_id = ?", userID).Exec(context.TODO())
	return err
}
