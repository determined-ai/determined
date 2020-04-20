package db

import (
	"crypto/ed25519"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
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
		return "", errors.Wrap(err, "failed to generate authentication token")
	}
	return token, nil
}

// UserByToken returns a user session given an authentication token.
func (db *PgDB) UserByToken(token string) (*model.User, *model.UserSession, error) {
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

// DeleteSessionByID deletes the session with the given ID.
func (db *PgDB) DeleteSessionByID(sessionID model.SessionID) error {
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

	var userID model.UserID
	if err := stmt.QueryRowx(user).Scan(&userID); err != nil {
		if pgerr, ok := errors.Cause(err).(*pq.Error); ok {
			if pgerr.Code == uniqueViolation {
				return 0, ErrDuplicateRecord
			}
		}
		return 0, errors.Wrapf(err, "error creating user %v", err)
	}

	return userID, nil
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

// AddUser creates a new user without a password.
func (db *PgDB) AddUser(user *model.User, ug *model.AgentUserGroup) error {
	tx, err := db.sql.Beginx()
	if err != nil {
		return errors.WithStack(err)
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
		return err
	}

	if ug != nil {
		if err := addAgentUserGroup(tx, userID, ug); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.WithStack(err)
	}

	tx = nil
	return nil
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
			"UPDATE users %v WHERE username = :username",
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

// UserByID returns the full user for a given ID.
func (db *PgDB) UserByID(userID model.UserID) (*model.FullUser, error) {
	var fu model.FullUser
	if err := db.query(`
SELECT
	u.id, u.username, u.admin, u.active,
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
