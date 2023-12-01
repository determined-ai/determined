package user

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

var scimLock sync.Mutex

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

	if ext.Validate(claims) != nil {
		return nil, nil, jwt.ErrTokenExpired
	}

	var isAdmin bool
	orgRoles, ok := claims.OrgRoles[ext.OrgID]
	if !ok || orgRoles.Role == model.NoRole {
		return nil, nil, db.ErrNotFound
	}
	if orgRoles.Role == model.AdminRole {
		isAdmin = true
	} else {
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
	}

	scimLock.Lock()
	defer scimLock.Unlock()

	scimUser, err := scimUserByAttribute(ctx, "user_id", claims.UserID)
	var u *model.User
	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			return nil, nil, err
		}

		// An existing SCIM user was not found: create or finish creating one.

		scimUser = &model.SCIMUser{
			ExternalID: claims.UserID,
			Emails:     model.SCIMEmailsFromJWT(claims),
			Name:       model.SCIMNameFromJWT(claims),
			RawAttributes: map[string]interface{}{
				"user_id": claims.UserID,
			},
			Username: claims.Email,
		}

		// Check for the temporary case where their email exists in users but no SCIM user exists
		u, err = ByUsername(ctx, claims.Email)
		if err != nil {
			if err != db.ErrNotFound {
				return nil, nil, err
			}

			// Legacy user was not found, so creating...
			_, err = AddSCIMUser(ctx, scimUser)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}
			u, err = UserBySCIMAttribute(ctx, "user_id", claims.UserID)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}
		} else {
			// Legacy user was found, so retrofit it...
			_, err = retrofitSCIMUser(ctx, scimUser, u.ID)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}
		}
	} else {
		// Existing SCIM user was found: retrieve or update all details.

		u, err = UserBySCIMAttribute(ctx, "user_id", claims.UserID)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		scimUser.Emails = model.SCIMEmailsFromJWT(claims)
		scimUser.Name = model.SCIMNameFromJWT(claims)
		scimUser.Username = claims.Email

		_, err = SetSCIMUser(ctx, scimUser.ID.String(), scimUser)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		u.Username = claims.Email
		u.Admin = isAdmin
		u.Active = true

		err = Update(ctx, u, []string{"username", "admin", "active"}, nil)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}

	u = &model.User{
		ID:           u.ID,
		Username:     claims.Email,
		PasswordHash: null.NewString("", false),
		Admin:        isAdmin,
		Active:       true,
	}

	session := &model.UserSession{
		ID:     model.SessionID(u.ID),
		UserID: u.ID,
		Expiry: time.Unix(claims.ExpiresAt, 0),
	}

	return u, session, nil
}
