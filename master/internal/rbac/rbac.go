package rbac

import (
	"database/sql"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/usergroup"
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
		Id:       rbacv1.PermissionType(int32(p.ID)),
		Name:     p.Name,
		IsGlobal: p.Global,
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

// Proto converts a Role into a RoleWithAssignments.
func (r *Role) Proto() *rbacv1.RoleWithAssignments {
	userAssignments, groupAssignments := RoleAssignments(r.RoleAssignments).Proto()

	return &rbacv1.RoleWithAssignments{
		Role: &rbacv1.Role{
			RoleId:      int32(r.ID),
			Name:        r.Name,
			Permissions: Permissions(r.Permissions).Proto(),
		},
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
		result = append(result, r.Proto())
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
	Group *usergroup.Group     `bun:"rel:has-one,join:group_id=id"`
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
				RoleId:      int32(a.RoleID),
				Name:        a.Role.Name,
				Permissions: Permissions(a.Role.Permissions).Proto(),
			}
		}

		var scopeWorkspaceID *wrappers.Int32Value
		if a.Scope != nil && a.Scope.WorkspaceID.Valid {
			scopeWorkspaceID = wrapperspb.Int32(a.Scope.WorkspaceID.Int32)
		}

		if a.Group.OwnerID == 0 {
			groupAssignments = append(groupAssignments, &rbacv1.GroupRoleAssignment{
				GroupId: int32(a.GroupID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role:             protoRole,
					ScopeWorkspaceId: scopeWorkspaceID,
				},
			})
		} else {
			userAssignments = append(userAssignments, &rbacv1.UserRoleAssignment{
				UserId: int32(a.Group.OwnerID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role:             protoRole,
					ScopeWorkspaceId: scopeWorkspaceID,
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
