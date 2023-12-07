package rbac

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

func init() {
	// Register many to many model so bun can better recognize m2m relation.
	db.RegisterModel((*RoleAssignment)(nil))
	db.RegisterModel((*PermissionAssignment)(nil))
}

// Permission represents a Permission as it's stored in the database.
type Permission struct {
	bun.BaseModel `bun:"table:permissions"`

	ID     int    `bun:"id,pk"`
	Name   string `bun:"name"`
	Global bool   `bun:"global_only"`
}

// Proto turns a permission into its rbac representation.
func (p *Permission) Proto() *rbacv1.Permission {
	return &rbacv1.Permission{
		Id:            rbacv1.PermissionType(int32(p.ID)),
		Name:          p.Name,
		ScopeTypeMask: p.ScopeTypeMask(),
	}
}

// ScopeTypeMask returns a mask of allowed scope types for this permission.
func (p *Permission) ScopeTypeMask() *rbacv1.ScopeTypeMask {
	return &rbacv1.ScopeTypeMask{
		Cluster:   true,
		Workspace: !p.Global,
	}
}

// Permissions is a list of permissions.
type Permissions []Permission

// IDs plucks the ids out of the permissions and returns them.
func (p Permissions) IDs() []int {
	if len(p) == 0 {
		return nil
	}

	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}

	return ids
}

// Proto turns a Permissions object into a list of rbac representations.
func (p Permissions) Proto() []*rbacv1.Permission {
	result := make([]*rbacv1.Permission, 0, len(p))

	for i := range p {
		result = append(result, p[i].Proto())
	}

	return result
}

// ScopeTypeMask returns a rolled-up mask of allowed scope types.
func (p Permissions) ScopeTypeMask() *rbacv1.ScopeTypeMask {
	workspace := true
	for i := range p {
		if !p[i].ScopeTypeMask().Workspace {
			workspace = false
			break
		}
	}

	return &rbacv1.ScopeTypeMask{
		Cluster:   true,
		Workspace: workspace,
	}
}

// PermissionAssignment contains the database representation of a PermissionAssignment
// as well as the Permission itself and the Role it is assigned to.
type PermissionAssignment struct {
	bun.BaseModel `bun:"table:permission_assignments"`

	PermissionID int `bun:",pk"`
	RoleID       int `bun:",pk"`

	Permission *Permission `bun:"rel:belongs-to,join:permission_id=id"`
	Role       *Role       `bun:"rel:belongs-to,join:role_id=id"`
}

// Role contains the database representation of a Role, along with
// the Permissions and RoleAssignments the Role references.
type Role struct {
	bun.BaseModel `bun:"table:roles,alias:roles"`

	ID              int               `bun:"id,pk,autoincrement" json:"id"`
	Name            string            `bun:"role_name,notnull" json:"name"`
	Created         time.Time         `bun:"created_at,notnull" json:"created"`
	Permissions     []Permission      `bun:"m2m:permission_assignments,join:Role=Permission"`
	RoleAssignments []*RoleAssignment `bun:"rel:has-many,join:id=role_id"`
}

// Proto converts a Role into a rbacv1.Role.
func (r *Role) Proto() *rbacv1.Role {
	return &rbacv1.Role{
		RoleId:        int32(r.ID),
		Name:          r.Name,
		Permissions:   Permissions(r.Permissions).Proto(),
		ScopeTypeMask: Permissions(r.Permissions).ScopeTypeMask(),
	}
}

// ProtoRoleWithAssignments converts a Role into a RoleWithAssignments.
func (r *Role) ProtoRoleWithAssignments() *rbacv1.RoleWithAssignments {
	userAssignments, groupAssignments := RoleAssignments(r.RoleAssignments).Proto()

	return &rbacv1.RoleWithAssignments{
		Role:                 r.Proto(),
		GroupRoleAssignments: groupAssignments,
		UserRoleAssignments:  userAssignments,
	}
}

// Roles is a list of Role.
type Roles []Role

// Proto converts Roles to a list of RoleWithAssignments.
func (rs Roles) Proto() []*rbacv1.RoleWithAssignments {
	result := make([]*rbacv1.RoleWithAssignments, 0, len(rs))
	for _, r := range rs {
		result = append(result, r.ProtoRoleWithAssignments())
	}

	return result
}

// RoleAssignment contains the database representation of RoleAssignment
// along with the Role, Group, and Scope that the RoleAssignment references.
type RoleAssignment struct {
	bun.BaseModel `bun:"table:role_assignments,alias:role_assignments"`

	GroupID int `bun:"group_id,pk" json:"group_id"`
	RoleID  int `bun:"role_id,pk" json:"role_id"`
	ScopeID int `bun:"scope_id,pk" json:"scope_id"`

	Role  *Role                `bun:"rel:belongs-to,join:role_id=id"`
	Group *model.Group         `bun:"rel:has-one,join:group_id=id"`
	Scope *RoleAssignmentScope `bun:"rel:has-one,join:scope_id=id"`
}

// RoleAssignments is a list of RoleAssignment.
type RoleAssignments []*RoleAssignment

// Proto converts a RoleAssignment into UserRoleAssignnments and GroupRoleAssignments.
func (ra RoleAssignments) Proto() ([]*rbacv1.UserRoleAssignment, []*rbacv1.GroupRoleAssignment) {
	var (
		userAssignments  []*rbacv1.UserRoleAssignment
		groupAssignments []*rbacv1.GroupRoleAssignment
	)

	for _, a := range ra {
		protoRole := &rbacv1.Role{RoleId: int32(a.RoleID)}
		if a.Role != nil {
			protoRole = &rbacv1.Role{
				RoleId:        int32(a.RoleID),
				Name:          a.Role.Name,
				Permissions:   Permissions(a.Role.Permissions).Proto(),
				ScopeTypeMask: Permissions(a.Role.Permissions).ScopeTypeMask(),
			}
		}

		var scopeWorkspaceID *int32
		if a.Scope != nil && a.Scope.WorkspaceID.Valid {
			scopeWorkspaceID = &a.Scope.WorkspaceID.Int32
		}

		if a.Group.OwnerID == 0 {
			groupAssignments = append(groupAssignments, &rbacv1.GroupRoleAssignment{
				GroupId: int32(a.GroupID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role:             protoRole,
					ScopeWorkspaceId: scopeWorkspaceID,
					ScopeCluster:     scopeWorkspaceID == nil,
				},
			})
		} else {
			userAssignments = append(userAssignments, &rbacv1.UserRoleAssignment{
				UserId: int32(a.Group.OwnerID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role:             protoRole,
					ScopeWorkspaceId: scopeWorkspaceID,
					ScopeCluster:     scopeWorkspaceID == nil,
				},
			})
		}
	}

	return userAssignments, groupAssignments
}

// RoleAssignmentScope represents a RoleAssignmentScope as it's stored in the database.
type RoleAssignmentScope struct {
	bun.BaseModel `bun:"table:role_assignment_scopes"`

	ID          int           `bun:"id,pk,autoincrement" json:"id"`
	WorkspaceID sql.NullInt32 `bun:"scope_workspace_id"  json:"workspace_id"`
}

// PermittedScopes returns a set of scopes that the user has the given permission on.
func PermittedScopes(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
	permission rbacv1.PermissionType,
) (model.AccessScopeSet, error) {
	returnScope := model.AccessScopeSet{}
	var workspaces []int

	// check if user has global permissions
	err := db.DoesPermissionMatch(ctx, curUser.ID, nil, permission)
	if err == nil {
		if requestedScope == 0 {
			err = db.Bun().NewSelect().Table("workspaces").Column("id").Scan(ctx, &workspaces)
			if err != nil {
				return nil, errors.Wrapf(err, "error getting workspaces from db")
			}

			for _, workspaceID := range workspaces {
				returnScope[model.AccessScopeID(workspaceID)] = true
			}
			return returnScope, nil
		}
		return model.AccessScopeSet{requestedScope: true}, nil
	}

	// get all workspaces user has permissions to
	workspaces, err = db.GetNonGlobalWorkspacesWithPermission(
		ctx, curUser.ID, permission)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting workspaces from db")
	}

	if requestedScope == 0 {
		for _, workspaceID := range workspaces {
			returnScope[model.AccessScopeID(workspaceID)] = true
		}
		return returnScope, nil
	}

	for _, workspaceID := range workspaces {
		if requestedScope == model.AccessScopeID(workspaceID) {
			return model.AccessScopeSet{requestedScope: true}, nil
		}
	}
	return model.AccessScopeSet{}, nil
}

// CheckForPermissionOptions represents the options for CheckForPermission.
type CheckForPermissionOptions struct {
	LogResult bool
}

// CheckForPermissionOptionsFunc is a function type for defining options for CheckForPermission.
type CheckForPermissionOptionsFunc func(*CheckForPermissionOptions)

// EnablePermissionCheckLogging enables or disables rbac audit logging for CheckForPermissons.
func EnablePermissionCheckLogging(flag bool) CheckForPermissionOptionsFunc {
	return func(o *CheckForPermissionOptions) {
		o.LogResult = flag
	}
}

// CheckForPermission checks if the user has the given permission on the given subject
// and logs the result unless logging is disabled.
func CheckForPermission(
	ctx context.Context, subject string, curUser *model.User,
	workspaceID *model.AccessScopeID, permission rbacv1.PermissionType,
	options ...CheckForPermissionOptionsFunc,
) (permErr error, err error) {
	// defaults to logging results.
	opts := &CheckForPermissionOptions{
		LogResult: true,
	}

	for _, option := range options {
		option(opts)
	}

	if opts.LogResult {
		fields := audit.ExtractLogFields(ctx)
		fields["userID"] = curUser.ID
		fields["username"] = curUser.Username
		fields["permissionsRequired"] = []audit.PermissionWithSubject{
			{
				PermissionTypes: []rbacv1.PermissionType{permission},
				SubjectType:     subject,
			},
		}
		defer func() {
			if err == nil {
				fields["permissionGranted"] = permErr == nil
				audit.Log(fields)
			}
		}()
	}

	var wid int32
	if workspaceID != nil {
		wid = int32(*workspaceID)
	}
	if err := db.DoesPermissionMatch(ctx, curUser.ID, &wid,
		permission); err != nil {
		switch typedErr := err.(type) {
		case authz.PermissionDeniedError:
			return typedErr, nil
		default:
			return nil, err
		}
	}
	return nil, nil
}
