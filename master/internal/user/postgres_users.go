package user

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserByID returns the full user for a given ID.
func UserByID(userID model.UserID) (*model.FullUser, error) {
	var fu model.FullUser
	query := `
SELECT
	u.id, u.username, u.display_name, u.admin, u.active,
	h.uid AS agent_uid, h.gid AS agent_gid, h.user_ AS agent_user, h.group_ AS agent_group
FROM users u
LEFT OUTER JOIN agent_user_groups h ON (u.id = h.user_id)
WHERE u.id = ?`
	if err := db.Bun().NewRaw(query, userID).Scan(context.Background(), &fu); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, db.ErrNotFound
		}

		return nil, err
	}

	return &fu, nil
}

// UserByUsername looks up a user by name in the database.
func UserByUsername(username string) (*model.User, error) {
	var user model.User
	username = strings.ToLower(username)
	err := db.Bun().NewSelect().Model(&user).
		Where("username = ?", username).Scan(context.Background())
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, db.ErrNotFound
		}

		return nil, err
	}
	return &user, nil
}

// SetDisplayName in User.
func SetDisplayName(userID int32, displayName *string) error {
	_, err := db.Bun().NewUpdate().Model((*model.User)(nil)).Set("display_name = ?", displayName).
		Where("id = ?", userID).Exec(context.TODO())
	return err
}

// AddUserExec execs an INSERT to create a new user.
func AddUserExec(user *model.User) error {
	ctx := context.TODO()
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		txErr := tx.Rollback()
		if txErr != nil && txErr != sql.ErrTxDone {
			log.WithError(txErr).Error("error rolling back transaction in AddUserExec")
		}
	}()

	_, err = tx.NewInsert().Model(user).ExcludeColumn("id").Returning("*").Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "error inserting user")
	}

	personalGroup := struct { // Duped definition to avoid import cycle. TODO redesign this.
		bun.BaseModel `bun:"table:groups,alias:groups"`

		ID      int          `bun:"id,pk,autoincrement" json:"id"`
		Name    string       `bun:"group_name,notnull"  json:"name"`
		OwnerID model.UserID `bun:"user_id,nullzero"    json:"userId,omitempty"`
	}{
		Name:    user.Username + db.PersonalGroupPostfix,
		OwnerID: user.ID,
	}
	if _, err = tx.NewInsert().Model(&personalGroup).Exec(ctx); err != nil {
		return errors.Wrap(err, "error inserting personal group")
	}

	groupMembership := struct {
		bun.BaseModel `bun:"table:user_group_membership"`

		UserID  model.UserID `bun:"user_id,notnull"`
		GroupID int          `bun:"group_id,notnull"`
	}{
		UserID:  user.ID,
		GroupID: personalGroup.ID,
	}
	if _, err = tx.NewInsert().Model(&groupMembership).Exec(ctx); err != nil {
		return errors.Wrap(err, "error adding user to personal group")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "error committing changes in AddUserExec")
	}

	return nil
}

// UserByExternalToken returns a user session derived from an external authentication token.
func UserByExternalToken(tokenText string,
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

	user, err := UserByUsername(claims.Email)
	if err != nil {
		if err != db.ErrNotFound {
			return nil, nil, err
		}
		user = &model.User{
			Username:     claims.Email,
			PasswordHash: null.NewString("", false),
			Admin:        true,
			Active:       true,
		}
		err := AddUserExec(user)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}

	session := &model.UserSession{
		ID:     model.SessionID(user.ID),
		UserID: user.ID,
		Expiry: time.Unix(claims.ExpiresAt, 0),
	}

	return user, session, nil
}

// UserByToken returns a user session given an authentication token.
func UserByToken(token string, ext *model.ExternalSessions) (
	*model.User, *model.UserSession, error,
) {
	if ext.JwtKey != "" {
		return UserByExternalToken(token, ext)
	}

	v2 := paseto.NewV2()

	var session model.UserSession
	err := v2.Verify(token, db.GetTokenKeys().PublicKey, &session, nil)
	if err != nil {
		return nil, nil, db.ErrNotFound
	}

	err = db.Bun().NewSelect().Model(&session).Where("id = ?", session.ID).Scan(context.Background())
	if err != nil {
		return nil, nil, err
	}

	if session.Expiry.Before(time.Now()) {
		return nil, nil, db.ErrNotFound
	}

	var user model.User
	query := `
	SELECT users.* FROM users
	JOIN user_sessions ON user_sessions.user_id = users.id
	WHERE user_sessions.id=?`

	err = db.Bun().NewRaw(query, session.ID).Scan(context.Background(), &user)
	if err != nil {
		return nil, nil, err
	}

	return &user, &session, nil
}
