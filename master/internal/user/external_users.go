package user

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ByExternalToken returns a user session derived from an external authentication token.
func ByExternalToken(ctx context.Context, tokenText string,
	ext *model.ExternalSessions,
) (*model.User, *model.UserSession, error) {
	token, err := jwt.ParseWithClaims(tokenText, &model.JWT{},
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
	claims := token.Claims.(*model.JWT)

	var isAdmin bool
	orgRoles, ok := claims.OrgRoles[ext.OrgID]
	if !ok || orgRoles.Role == model.NoRole {
		return nil, nil, db.ErrNotFound
	}
	clusterAccess, ok := orgRoles.ClusterRoles[ext.ClusterID]
	if ok {
		if clusterAccess == model.NoRole {
			return nil, nil, db.ErrNotFound
		}
		isAdmin = clusterAccess == model.AdminRole
	} else {
		if orgRoles.DefaultClusterRole == model.NoRole {
			return nil, nil, db.ErrNotFound
		}
		isAdmin = orgRoles.DefaultClusterRole == model.AdminRole
	}

	user, err := UserByUsername(claims.Email)
	if err != nil {
		if err != db.ErrNotFound {
			return nil, nil, err
		}
		user = &model.User{
			Username:     claims.Email,
			PasswordHash: null.NewString("", false),
			Admin:        isAdmin,
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
