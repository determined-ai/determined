package db

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

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
	privateKey := db.tokenKeys.PrivateKey
	token, err := v2.Sign(privateKey, userSession, nil)
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

// DeleteUserSessionByID deletes the user session with the given ID.
func (db *PgDB) DeleteUserSessionByID(sessionID model.SessionID) error {
	_, err := db.sql.Exec("DELETE FROM user_sessions WHERE id=$1", sessionID)
	return err
}

func addUser(tx *sqlx.Tx, user *model.User) (model.UserID, error) {
	stmt, err := tx.PrepareNamed(`
INSERT INTO users
(username, admin, active, password_hash, display_name, remote)
VALUES (:username, :admin, :active, :password_hash, :display_name, :remote)
RETURNING id`)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer stmt.Close()

	if err := stmt.QueryRowx(user).Scan(&user.ID); err != nil {
		if pgerr, ok := errors.Cause(err).(*pgconn.PgError); ok {
			if pgerr.Code == CodeUniqueViolation {
				return 0, ErrDuplicateRecord
			}
		}
		return 0, errors.Wrapf(err, "error creating user %v", err)
	}

	if err := addUserPersonalGroup(tx, user.ID); err != nil {
		return 0, errors.Wrap(err, "error adding users personal group")
	}

	return user.ID, nil
}

// PersonalGroupPostfix is the system postfix appended to the username of all personal groups.
const PersonalGroupPostfix = "DeterminedPersonalGroup"

func addUserPersonalGroup(tx *sqlx.Tx, userID model.UserID) error {
	query := `
INSERT INTO groups(group_name, user_id)
SELECT $1 || $3 AS group_name, id AS user_id FROM users
WHERE id = $2`
	if _, err := tx.Exec(query, strconv.Itoa(int(userID)), userID, PersonalGroupPostfix); err != nil {
		return errors.WithStack(err)
	}

	query = `
INSERT INTO user_group_membership(user_id, group_id)
SELECT user_id AS user_id, id AS group_id FROM groups
WHERE user_id = $1`
	if _, err := tx.Exec(query, userID); err != nil {
		return errors.WithStack(err)
	}

	return nil
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
		if *ug != (model.AgentUserGroup{}) {
			if err = addAgentUserGroup(tx, updated.ID, ug); err != nil {
				return err
			}
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
	setTokenKeys(db.tokenKeys)
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

// OverwriteUserSetting resets user settings and fills them with passed values.
func OverwriteUserSetting(userID model.UserID, settings []*userv1.UserWebSetting) error {
	err := ResetUserSetting(userID)
	for _, v := range settings {
		userSetting := model.UserWebSetting{
			UserID:      userID,
			Key:         v.Key,
			Value:       v.Value,
			StoragePath: v.StoragePath,
		}
		err = UpdateUserSetting(&userSetting)
	}
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
