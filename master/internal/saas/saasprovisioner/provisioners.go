package saasprovisioner

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/internal/saas"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/config"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/model"
)

var scimLock sync.Mutex

// SAASProvisioner is a struct for provisioning saas users.
type SAASProvisioner struct{}

func provisionUserRBAC(ctx context.Context, scimUser *model.SCIMUser, clusterRole model.Role) error {
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// AddSCIMUser calls AddUserTx, which creates the user's personal group
		// We will probably want to get the group ID back, or we can just search for it
		// Optimistically hardcoding for now for testing purposes.
		scimUser, err := user.AddSCIMUser(ctx, scimUser)
		if err != nil {
			return err
		}
		personalGroupName := fmt.Sprintf("%d%s", scimUser.UserID, user.PersonalGroupPostfix)
		groups, _, _, err := usergroup.SearchGroups(ctx, personalGroupName, scimUser.UserID, 0, 1)
		if len(groups) < 1 {
			return fmt.Errorf("error finding group id for new user with id %d: %w", scimUser.UserID, err)
		}
		personalGroupID := groups[0].ID

		clusterRoleID := 3 // WorkspaceCreator
		if clusterRole == model.AdminRole {
			clusterRoleID = 1 // ClusterAdmin
		}

		groupRoleAssignment := &rbacv1.GroupRoleAssignment{
			GroupId: int32(personalGroupID),
			RoleAssignment: &rbacv1.RoleAssignment{
				Role: &rbacv1.Role{
					RoleId: int32(clusterRoleID),
				},
				ScopeCluster: true,
				// ScopeWorkspaceId: &scopeID,
			},
		}
		err = rbac.AddGroupAssignmentsTx(ctx, tx, []*rbacv1.GroupRoleAssignment{groupRoleAssignment})
		return err
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func provisionUserBasic(ctx context.Context, scimUser *model.SCIMUser) error {
	// Legacy user was not found, so creating...
	_, err := user.AddSCIMUser(ctx, scimUser)
	return err
}

// GetAndMaybeProvisionUserByToken returns a user session derived from an external authentication token.
func (p *SAASProvisioner) GetAndMaybeProvisionUserByToken(ctx context.Context, tokenText string,
	ext *model.ExternalSessions,
) (*model.User, *model.UserSession, error) {
	rbacEnabled := config.GetAuthZConfig().IsRBACEnabled()
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
		return nil, nil, fmt.Errorf("error parsing jwt: %w", err)
	}

	claims := token.Claims.(*model.JWT)

	if ext.Validate(claims) != nil {
		return nil, nil, jwt.ErrTokenExpired
	}

	var clusterRole model.Role
	active := false

	orgRoles, ok := claims.OrgRoles[ext.OrgID]
	if !ok || orgRoles.Role == model.NoRole {
		return nil, nil, db.ErrNotFound
	}

	active, ok = orgRoles.ClusterActivations[ext.ClusterID]
	if !ok {
		active = true
	}

	if orgRoles.Role == model.AdminRole {
		clusterRole = model.AdminRole
	} else {
		clusterRole = orgRoles.DefaultClusterRole
	}

	if clusterRole == model.NoRole {
		return nil, nil, db.ErrNotFound
	}

	scimLock.Lock()
	defer scimLock.Unlock()

	scimUser, err := user.ScimUserByAttribute(ctx, "user_id", claims.UserID)
	var u *model.User
	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			return nil, nil, err
		}

		// An existing SCIM user was not found: create or finish creating one.
		scimUser = &model.SCIMUser{
			Active:     active,
			ExternalID: claims.UserID,
			Emails:     model.SCIMEmailsFromJWT(claims),
			Name:       model.SCIMNameFromJWT(claims),
			RawAttributes: map[string]interface{}{
				"user_id": claims.UserID,
			},
			Username: claims.Email,
		}

		// Check for the temporary case where their email exists in users but no SCIM user exists
		u, err = user.ByUsername(ctx, claims.Email)
		if err != nil {
			if !errors.Is(err, db.ErrNotFound) {
				return nil, nil, err
			}

			if rbacEnabled {
				err = provisionUserRBAC(ctx, scimUser, clusterRole)
			} else {
				err = provisionUserBasic(ctx, scimUser)
			}

			if err != nil {
				return nil, nil, errors.WithStack(err)
			}
			u, err = user.UserBySCIMAttribute(ctx, "user_id", claims.UserID)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}
		} else {
			// Legacy user was found, so retrofit it...
			_, err = user.RetrofitSCIMUser(ctx, scimUser, u.ID)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}
		}
	} else {
		// Existing SCIM user was found: retrieve or update all details.

		u, err = user.UserBySCIMAttribute(ctx, "user_id", claims.UserID)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		scimUser.Emails = model.SCIMEmailsFromJWT(claims)
		scimUser.Name = model.SCIMNameFromJWT(claims)
		scimUser.Username = claims.Email

		_, err = user.SetSCIMUser(ctx, scimUser.ID.String(), scimUser)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		u.Username = claims.Email
		u.Admin = clusterRole == model.AdminRole
		u.Active = active

		err = user.Update(ctx, u, []string{"username", "admin", "active"}, nil)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}

	u = &model.User{
		ID:           u.ID,
		Username:     claims.Email,
		PasswordHash: null.NewString("", false),
		Admin:        clusterRole == model.AdminRole,
		Active:       active,
	}

	session := &model.UserSession{
		ID:     model.SessionID(u.ID),
		UserID: u.ID,
		Expiry: time.Unix(claims.ExpiresAt, 0),
	}

	return u, session, nil
}

// Register initialized the external user provisioner.
func Register() {
	saas.RegisterProvisioner("saas", &SAASProvisioner{})
}
