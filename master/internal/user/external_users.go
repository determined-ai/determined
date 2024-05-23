package user

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/db"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
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

	orgRoles, ok := claims.OrgRoles[ext.OrgID]
	if !ok || orgRoles.Role == model.NoRole {
		return nil, nil, db.ErrNotFound
	}

	clusterRole := model.NoRole

	if orgRoles.Role == model.AdminRole || orgRoles.DefaultClusterRole == model.AdminRole {
		clusterRole = model.AdminRole
	} else {
		if orgRoles.DefaultClusterRole == model.UserRole {
			clusterRole = model.UserRole
		}
	}
	isAdmin := clusterRole == model.AdminRole

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
			if !errors.Is(err, db.ErrNotFound) {
				return nil, nil, err
			}

			// Legacy user was not found, so creating...
			err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				// AddSCIMUser calls AddUserTx, which creates the user's personal group
				// We will probably want to get the group ID back, or we can just search for it
				// Optimistically hardcoding for now for testing purposes.
				_, err = AddSCIMUser(ctx, scimUser)
				if err != nil {
					return err
				}

				if clusterRole != model.NoRole {

					clusterRoleID := 3 // WorkspaceCreator
					if isAdmin {
						clusterRoleID = 1 // ClusterAdmin
					}

					// TODO: FIX
					personalGroupId := 5
					// scopeID := 1

					groupRoleAssignment := &rbacv1.GroupRoleAssignment{
						GroupId: int32(personalGroupId),
						RoleAssignment: &rbacv1.RoleAssignment{
							Role: &rbacv1.Role{
								RoleId: int32(clusterRoleID),
							},
							ScopeCluster: true,
							// ScopeWorkspaceId: &scopeID,
						},
					}
					AddGroupAssignmentsTx(ctx, tx, []*rbacv1.GroupRoleAssignment{groupRoleAssignment})
				}
				return nil
			})
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

func whichAreGlobalOnly(ctx context.Context, idb bun.IDB, roles []int32) ([]int32, error) {
	if len(roles) < 1 {
		return nil, nil
	}

	if idb == nil {
		idb = db.Bun()
	}

	var results []int32
	err := idb.NewSelect().Distinct().
		Column("role_id").
		TableExpr("permission_assignments AS pa").
		Join("JOIN permissions AS p ON pa.permission_id=p.id").
		Where("p.global_only AND pa.role_id IN (?)", bun.In(roles)).
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func enforceGlobalOnly(ctx context.Context, idb bun.IDB,
	assignments []*rbacv1.GroupRoleAssignment,
) (bool, error) {
	var toBeLocallyAssigned []int32
	for _, a := range assignments {
		if a.RoleAssignment.ScopeWorkspaceId != nil {
			toBeLocallyAssigned = append(toBeLocallyAssigned, a.RoleAssignment.Role.RoleId)
		}
	}

	globalOnly, err := whichAreGlobalOnly(ctx, idb, toBeLocallyAssigned)
	if err != nil {
		return false, errors.Wrap(db.MatchSentinelError(err),
			"error checking global-only permissions were only being assigned globally")
	}

	if len(globalOnly) > 0 {
		return false, nil
	}

	return true, nil
}

type RoleAssignment struct {
	bun.BaseModel `bun:"table:role_assignments,alias:role_assignments"`

	GroupID int `bun:"group_id,pk" json:"group_id"`
	RoleID  int `bun:"role_id,pk" json:"role_id"`
	ScopeID int `bun:"scope_id,pk" json:"scope_id"`

	Role  *Role                `bun:"rel:belongs-to,join:role_id=id"`
	Group *model.Group         `bun:"rel:has-one,join:group_id=id"`
	Scope *RoleAssignmentScope `bun:"rel:has-one,join:scope_id=id"`
}

type Role struct {
	bun.BaseModel `bun:"table:roles,alias:roles"`

	ID              int               `bun:"id,pk,autoincrement" json:"id"`
	Name            string            `bun:"role_name,notnull" json:"name"`
	Created         time.Time         `bun:"created_at,notnull" json:"created"`
	Permissions     []Permission      `bun:"m2m:permission_assignments,join:Role=Permission"`
	RoleAssignments []*RoleAssignment `bun:"rel:has-many,join:id=role_id"`
}
type Permission struct {
	bun.BaseModel `bun:"table:permissions"`

	ID     int    `bun:"id,pk"`
	Name   string `bun:"name"`
	Global bool   `bun:"global_only"`
}

var ErrGlobalAssignedLocally = errors.New("a global-only permission cannot be assigned to a local scope")

// AddGroupAssignmentsTx adds a role assignment to a group while inside a transaction.
func AddGroupAssignmentsTx(ctx context.Context, idb bun.IDB, groups []*rbacv1.GroupRoleAssignment,
) error {
	if len(groups) < 1 {
		return nil
	}

	if idb == nil {
		idb = db.Bun()
	}

	valid, err := enforceGlobalOnly(ctx, idb, groups)
	if err != nil {
		return err
	} else if !valid {
		return ErrGlobalAssignedLocally
	}

	for _, group := range groups {
		s, err := getOrCreateRoleAssignmentScopeTx(ctx, idb, group.RoleAssignment)
		if err != nil {
			return errors.Wrapf(db.MatchSentinelError(err),
				"Error getting scope for group id %d", group.GroupId)
		}

		roleAssignment := RoleAssignment{
			GroupID: int(group.GroupId),
			RoleID:  int(group.RoleAssignment.Role.RoleId),
			ScopeID: s.ID,
		}

		// insert into role assignments
		_, err = idb.NewInsert().Model(&roleAssignment).Exec(ctx)
		if err != nil {
			return errors.Wrapf(db.MatchSentinelError(err),
				"Error inserting assignment for group id %d", group.GroupId)
		}
	}

	return nil
}

type RoleAssignmentScope struct {
	bun.BaseModel `bun:"table:role_assignment_scopes"`

	ID          int           `bun:"id,pk,autoincrement" json:"id"`
	WorkspaceID sql.NullInt32 `bun:"scope_workspace_id"  json:"workspace_id"`
}

func getOrCreateRoleAssignmentScopeTx(ctx context.Context, idb bun.IDB,
	assignment *rbacv1.RoleAssignment,
) (RoleAssignmentScope, error) {
	if idb == nil {
		idb = db.Bun()
	}

	r := RoleAssignmentScope{}

	scopeSelect := idb.NewSelect().Model(&r)

	// Postgres unique constraints do not block duplicate null values
	// so we must check if a null scope already exists
	if assignment.ScopeWorkspaceId == nil {
		scopeSelect = scopeSelect.Where("scope_workspace_id IS NULL")
		err := scopeSelect.Scan(ctx)

		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return r, errors.Wrapf(db.MatchSentinelError(err), "Error checking for a null workspace")
		} else if err == nil {
			return r, nil
		}
	} else {
		scopeSelect = scopeSelect.Where("scope_workspace_id = ?", *assignment.ScopeWorkspaceId)

		r.WorkspaceID.Int32 = *assignment.ScopeWorkspaceId
		r.WorkspaceID.Valid = true
	}

	// Try to insert RoleAssignmentScope, do nothing if it already exists in the table
	_, err := idb.NewInsert().Model(&r).Ignore().Exec(ctx)
	if err != nil {
		return r, errors.Wrapf(db.MatchSentinelError(err), "Error creating a RoleAssignmentScope")
	}

	// Retrieve the role assignment scope from DB
	err = scopeSelect.Scan(ctx)
	if err != nil {
		return r, errors.Wrapf(db.MatchSentinelError(err), "Error getting RoleAssignmentScope %d", r.ID)
	}

	return r, nil
}
