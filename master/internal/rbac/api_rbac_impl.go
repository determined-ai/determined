package rbac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

func init() {
	rbacAPIServer = &RBACAPIServerImpl{}
}

// RBACAPIServerImpl contains the RBAC implementation of RBACAPIServer.
type RBACAPIServerImpl struct{}

// GetPermissionsSummary gets a permission overview for the currently logged in user.
func (a *RBACAPIServerImpl) GetPermissionsSummary(
	ctx context.Context, req *apiv1.GetPermissionsSummaryRequest,
) (resp *apiv1.GetPermissionsSummaryResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	summary, err := GetPermissionSummary(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	var roles []Role
	var assignments []*rbacv1.RoleAssignmentSummary
	for role, roleAssignments := range summary {
		var workspaceIDs []int32
		isGlobal := false
		for _, assign := range roleAssignments {
			if assign.Scope.WorkspaceID.Valid {
				workspaceIDs = append(workspaceIDs, assign.Scope.WorkspaceID.Int32)
			} else {
				isGlobal = true
			}
		}

		assignments = append(assignments, &rbacv1.RoleAssignmentSummary{
			RoleId:            int32(role.ID),
			ScopeWorkspaceIds: workspaceIDs,
			IsGlobal:          isGlobal,
		})
		roles = append(roles, *role)
	}

	return &apiv1.GetPermissionsSummaryResponse{
		Roles:       dbRolesToAPISummary(roles),
		Assignments: assignments,
	}, nil
}

// GetRolesByID searches for roles that fulfill the criteria given by the user.
func (a *RBACAPIServerImpl) GetRolesByID(ctx context.Context, req *apiv1.GetRolesByIDRequest,
) (resp *apiv1.GetRolesByIDResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	roles, err := GetRolesByIDs(ctx, req.RoleIds...)
	if err != nil {
		return nil, err
	}

	if len(roles) != len(req.RoleIds) {
		return nil, errNotFound
	}

	response := apiv1.GetRolesByIDResponse{
		Roles: roles,
	}

	return &response, nil
}

// GetRolesAssignedToUser retrieves all the roles assigned to the user or to the groups the
// user belongs in.
func (a *RBACAPIServerImpl) GetRolesAssignedToUser(ctx context.Context,
	req *apiv1.GetRolesAssignedToUserRequest,
) (resp *apiv1.GetRolesAssignedToUserResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	if req.UserId == 0 {
		return nil, errBadRequest
	}

	groups, _, _, err := usergroup.SearchGroups(ctx, "", model.UserID(req.UserId), 0, 0)
	if err != nil {
		return nil, err
	}

	groupIDs := make([]int32, len(groups))
	for i := range groups {
		groupIDs[i] = int32(groups[i].ID)
	}

	roles, err := GetRolesAssignedToGroupsTx(ctx, nil, groupIDs...)
	if err != nil {
		return nil, err
	}

	return &apiv1.GetRolesAssignedToUserResponse{
		Roles: dbRolesToAPISummary(roles),
	}, nil
}

// GetRolesAssignedToGroup gets the roles belonging to a group.
func (a *RBACAPIServerImpl) GetRolesAssignedToGroup(ctx context.Context,
	req *apiv1.GetRolesAssignedToGroupRequest) (resp *apiv1.GetRolesAssignedToGroupResponse,
	err error,
) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	roles, err := GetRolesAssignedToGroupsTx(ctx, nil, req.GroupId)
	if err != nil {
		return nil, err
	}

	return &apiv1.GetRolesAssignedToGroupResponse{
		Roles: dbRolesToAPISummary(roles),
	}, nil
}

// SearchRolesAssignableToScope looks for roles we can add to the scope.
func (a *RBACAPIServerImpl) SearchRolesAssignableToScope(ctx context.Context,
	req *apiv1.SearchRolesAssignableToScopeRequest) (_ *apiv1.SearchRolesAssignableToScopeResponse,
	err error,
) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	if req.Limit == 0 {
		req.Limit = maxLimit
	}

	roles, tableTotal, err := GetAllRoles(ctx, req.WorkspaceId != nil, int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, err
	}

	return &apiv1.SearchRolesAssignableToScopeResponse{
		Roles: dbRolesToAPISummary(roles),
		Pagination: &apiv1.Pagination{
			Offset:     req.Offset,
			Limit:      req.Limit,
			StartIndex: req.Offset,
			EndIndex:   req.Offset + int32(len(roles)),
			Total:      tableTotal,
		},
	}, nil
}

// ListRoles returns all roles.
func (a *RBACAPIServerImpl) ListRoles(ctx context.Context, req *apiv1.ListRolesRequest,
) (resp *apiv1.ListRolesResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	if req.Limit == 0 {
		req.Limit = maxLimit
	}

	roles, tableTotal, err := GetAllRoles(ctx, false, int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, err
	}

	return &apiv1.ListRolesResponse{
		Roles: dbRolesToAPISummary(roles),
		Pagination: &apiv1.Pagination{
			Offset:     req.Offset,
			Limit:      req.Limit,
			StartIndex: req.Offset,
			EndIndex:   req.Offset + int32(len(roles)),
			Total:      tableTotal,
		},
	}, nil
}

// AssignRoles grants the specified users or groups a particular role.
func (a *RBACAPIServerImpl) AssignRoles(ctx context.Context, req *apiv1.AssignRolesRequest,
) (resp *apiv1.AssignRolesResponse, err error) {
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	err = ensureGroupsAreNotPersonal(ctx, req.GroupRoleAssignments)
	if err != nil {
		return nil, err
	}

	err = AddRoleAssignments(ctx, req.GroupRoleAssignments, req.UserRoleAssignments)
	if err != nil {
		return nil, err
	}

	return &apiv1.AssignRolesResponse{}, nil
}

// RemoveAssignments removes the specified users or groups from a role.
func (a *RBACAPIServerImpl) RemoveAssignments(ctx context.Context,
	req *apiv1.RemoveAssignmentsRequest,
) (resp *apiv1.RemoveAssignmentsResponse, err error) {
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	err = ensureGroupsAreNotPersonal(ctx, req.GroupRoleAssignments)
	if err != nil {
		return nil, err
	}

	err = RemoveRoleAssignments(ctx, req.GroupRoleAssignments, req.UserRoleAssignments)
	if err != nil {
		return nil, err
	}

	return &apiv1.RemoveAssignmentsResponse{}, nil
}

func dbRolesToAPISummary(roles []Role) []*rbacv1.Role {
	apiRoles := make([]*rbacv1.Role, 0, len(roles))
	for _, r := range roles {
		apiRoles = append(apiRoles, &rbacv1.Role{
			RoleId:      int32(r.ID),
			Name:        r.Name,
			Permissions: Permissions(r.Permissions).Proto(),
		})
	}

	return apiRoles
}

const (
	maxLimit = 500
)

var (
	errBadRequest   = status.Error(codes.InvalidArgument, "bad request")
	errInvalidLimit = status.Errorf(codes.InvalidArgument,
		"Bad request: limit is required and must be <= %d", maxLimit)
	errNotFound        = status.Error(codes.NotFound, "not found")
	errDuplicateRecord = status.Error(codes.AlreadyExists, "duplicate record")
	errInternal        = status.Error(codes.Internal, "internal server error")
	errPassthroughMap  = map[error]bool{
		nil:                true,
		errBadRequest:      true,
		errInvalidLimit:    true,
		errNotFound:        true,
		errDuplicateRecord: true,
		errInternal:        true,
	}
)

func mapAndFilterErrors(err error) error {
	if allowed := errPassthroughMap[err]; allowed {
		return err
	}

	switch {
	case errors.Is(err, db.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, db.ErrDuplicateRecord):
		return status.Error(codes.AlreadyExists, err.Error())
	}

	logrus.WithError(err).Debug("suppressing error at API boundary")

	return errInternal
}

func ensureGroupsAreNotPersonal(ctx context.Context,
	assignments []*rbacv1.GroupRoleAssignment,
) error {
	if len(assignments) == 0 {
		return nil
	}

	groupIDs := groupIDsFromAssignments(assignments)

	// FIXME: do this in one query
	for _, gid := range groupIDs {
		group, err := usergroup.GroupByIDTx(ctx, nil, int(gid))
		if err != nil {
			return err
		}
		if group.OwnerID != 0 {
			return errBadRequest
		}
	}

	return nil
}

func groupIDsFromAssignments(assignments []*rbacv1.GroupRoleAssignment) []int32 {
	groupIDs := make([]int32, 0, len(assignments))
	for _, ra := range assignments {
		groupIDs = append(groupIDs, ra.GroupId)
	}
	return groupIDs
}
